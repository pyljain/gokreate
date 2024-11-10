package main

import (
	"gokreate/internal/actions"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "gk",
		Usage: "GoKreate lets you scaffold front to back apps in React and Golang",
		Commands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "Initialize a new GoKreate project",
				Action: actions.Init,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "project-name",
						Aliases:  []string{"prj"},
						Usage:    "Choose a name for your project",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "db",
						Aliases: []string{"d"},
						Usage: `Pass the environment variable that has the connection string
								  DB_CONNECTION_STRING`,
					},
					&cli.StringFlag{
						Name:    "frontend-port",
						Aliases: []string{"f"},
						Usage:   "Frontend port to use",
						Value:   "3000",
					},
					&cli.StringFlag{
						Name:    "backend-port",
						Aliases: []string{"b"},
						Usage:   "Backend port to use",
						Value:   "9081",
					},
				},
			},
			{
				Name:   "run",
				Usage:  "Run the project and watch for changes",
				Action: actions.Run,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
	// init <projectname> --db ENV_VARIABLE --frontend-port --backend-port
	// run
	// g db init
	// g db add_collection
}
