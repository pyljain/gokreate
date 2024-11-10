package actions

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/urfave/cli/v2"
)

func Init(ctx *cli.Context) error {
	// Clone a repository

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	directoryToCloneInto := path.Join(cwd, ctx.String("project-name"))
	_, err = git.PlainClone(directoryToCloneInto, false, &git.CloneOptions{
		URL:      "https://github.com/patnaikshekhar/gocreate.git",
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil {
		return err
	}

	// Remove git remote
	err = os.RemoveAll(path.Join(directoryToCloneInto, ".git"))
	if err != nil {
		return err
	}

	// Update templated values
	err = filepath.WalkDir(directoryToCloneInto, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".tmpl" {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		templ, err := template.New(path).Parse(string(contents))
		if err != nil {
			return err
		}

		newPath := path[:len(path)-5]

		f, err := os.Create(newPath)
		if err != nil {
			return err
		}
		defer f.Close()

		err = templ.Execute(f, map[string]string{
			"ProjectName":  ctx.String("project-name"),
			"BackendPort":  ctx.String("backend-port"),
			"FrontendPort": ctx.String("frontend-port"),
		})
		if err != nil {
			return err
		}

		err = os.Remove(path)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Execute make target
	cmd := exec.Command("make", "build-all")
	cmd.Dir = directoryToCloneInto
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
