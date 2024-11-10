package actions

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func Run(ctx *cli.Context) error {

	var backendCmd *exec.Cmd
	var mu sync.Mutex

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Println("Received interrupt signal, shutting down...")
		// Lock before accessing backendCmd.
		mu.Lock()
		if backendCmd != nil && backendCmd.Process != nil {
			pgid, err := syscall.Getpgid(backendCmd.Process.Pid)
			if err == nil {
				// Send SIGKILL to the process group to forcefully terminate.
				syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				// If getting pgid fails, kill the process directly.
				backendCmd.Process.Kill()
			}
		}
		mu.Unlock()
	}()

	eg, egCtx := errgroup.WithContext(ctx.Context)
	eg.Go(func() error {
		cmd := exec.CommandContext(egCtx, "make", "run-frontend")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			return err
		}
		return nil
	})

	var backendCtx context.Context
	var backendCancel context.CancelFunc
	wd, err := os.Getwd()
	log.Printf("Directory Name is %s", wd)
	if err != nil {
		return err
	}

	prj := filepath.Base(wd)

	eg.Go(func() error {
		for {
			mu.Lock()
			buildCmd := exec.CommandContext(egCtx, "make", "build-backend")
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			buildCmd.Stdin = os.Stdin
			err := buildCmd.Run()
			if err != nil {
				return err
			}

			log.Printf("Starting backend")
			backendCtx, backendCancel = context.WithCancel(egCtx)

			backendCmd = exec.CommandContext(backendCtx, fmt.Sprintf("./bin/%s", prj))
			backendCmd.Stdout = os.Stdout
			backendCmd.Stderr = os.Stderr
			backendCmd.Stdin = os.Stdin
			backendCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			err = backendCmd.Start()
			if err != nil {
				return err
			}
			mu.Unlock()

			// Goroutine to handle context cancellation
			go func() {
				<-backendCtx.Done()
				pgid, err := syscall.Getpgid(backendCmd.Process.Pid)
				if err == nil {
					// Send SIGTERM to the process group
					syscall.Kill(-pgid, syscall.SIGTERM)
				} else {
					// Fallback to killing the process directly
					backendCmd.Process.Kill()
				}
			}()

			err = backendCmd.Wait()
			if err != nil {
				if egCtx.Err() == nil && backendCtx.Err() != nil {
					// Process was terminated due to context cancellation
					continue
				}
				return err
			}
			return nil
		}
	})

	eg.Go(func() error {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		defer watcher.Close()

		err = addToWatchList(".", watcher)
		if err != nil {
			return err
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					log.Println("Created or modified file:", event.Name)
					err := addToWatchList(".", watcher)
					if err != nil {
						return err
					}
					if backendCancel != nil {
						backendCancel()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return err
				}
			case <-egCtx.Done():
				return nil
			}
		}
	})

	err = eg.Wait()
	return err
}

func addToWatchList(path string, watcher *fsnotify.Watcher) error {
	var directoriesToWatch []string
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {

		if d.Name() == "ui" {
			return fs.SkipDir
		}

		if d.IsDir() {
			directoriesToWatch = append(directoriesToWatch, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	wl := watcher.WatchList()
	for _, d := range directoriesToWatch {
		if slices.Contains(wl, d) {
			continue
		}

		err = watcher.Add(d)
		if err != nil {
			return err
		}

		log.Printf("Add direction: %s to watchlist", d)
	}

	return nil

}
