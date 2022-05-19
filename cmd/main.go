/*===========================================================================*\
 *           MIT License Copyright (c) 2022 Kris Nóva <kris@nivenly.com>     *
 *                                                                           *
 *                ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓                *
 *                ┃   ███╗   ██╗ ██████╗ ██╗   ██╗ █████╗   ┃                *
 *                ┃   ████╗  ██║██╔═████╗██║   ██║██╔══██╗  ┃                *
 *                ┃   ██╔██╗ ██║██║██╔██║██║   ██║███████║  ┃                *
 *                ┃   ██║╚██╗██║████╔╝██║╚██╗ ██╔╝██╔══██║  ┃                *
 *                ┃   ██║ ╚████║╚██████╔╝ ╚████╔╝ ██║  ██║  ┃                *
 *                ┃   ╚═╝  ╚═══╝ ╚═════╝   ╚═══╝  ╚═╝  ╚═╝  ┃                *
 *                ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛                *
 *                                                                           *
 *                       This machine kills fascists.                        *
 *                                                                           *
\*===========================================================================*/

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kris-nova/kush"

	"github.com/kris-nova/kush/pkg/kobfuscate"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var cfg = &AppOptions{}

type AppOptions struct {
	verbose bool
	name    string
}

func main() {
	/* Change version to -V */
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "The version of the program.",
	}
	app := &cli.App{
		Name:     kush.Name,
		Version:  kush.Version,
		Compiled: time.Now(),
		Authors: []*cli.Author{
			&cli.Author{
				Name:  kush.AuthorName,
				Email: kush.AuthorEmail,
			},
		},
		Copyright: kush.Copyright,
		HelpName:  kush.Copyright,
		Usage:     "kobfuscate - Kubernetes obfuscation tool.",
		UsageText: `kobfuscate <options> <flags>`,
		Commands: []*cli.Command{
			&cli.Command{},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Destination: &cfg.verbose,
				Usage:       "Toggle verbosity.",
			},
			&cli.StringFlag{
				Name:        "name",
				Aliases:     []string{"n"},
				Destination: &cfg.name,
				Usage:       "Name for the objects to share.",
				Value:       "kush",
			},
		},
		EnableBashCompletion: true,
		HideHelp:             false,
		HideVersion:          false,
		Before: func(c *cli.Context) error {
			return nil
		},
		After: func(c *cli.Context) error {
			return nil
		},
		Action: func(c *cli.Context) error {

			if cfg.verbose {
				logrus.SetLevel(logrus.DebugLevel)
			} else {
				logrus.SetLevel(logrus.InfoLevel)
			}
			logrus.Infof("Starting kobfuscate...")

			// By default, this system will do everything
			// it can to start a ksh shell!

			runtime := kobfuscate.NewRuntime("kush")
			logrus.Infof("Starting runtime...")
			err := runtime.EscapeInit()
			if err != nil {
				return fmt.Errorf("error initializing: %v", err)
			}
			err = runtime.Hide()
			if err != nil {
				return fmt.Errorf("unable to obfuscate from Kubernetes: %v", err)
			}
			return fmt.Errorf("unable to obfuscate from Kubernetes")
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Errorf("exec failure: %v", err)
	}
}
