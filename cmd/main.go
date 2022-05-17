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
	"os"
	"time"

	"github.com/kris-nova/kush"

	"github.com/kris-nova/kush/pkg/ksh"

	"github.com/kris-nova/kush/pkg/kobfuscate"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var cfg = &AppOptions{}

type AppOptions struct {
	verbose bool

	// clusterMode will terminate the program if kush is unable
	// to detect a Kubernetes cluster, or obfuscate itself.
	clusterMode bool
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
		Usage:     "kush - Kubernetes Unhinged Shell.",
		UsageText: `kush <options> <flags>`,
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
			&cli.BoolFlag{
				Name:        "cluster",
				Aliases:     []string{"x"},
				Destination: &cfg.clusterMode,
				Usage:       "Toggle cluster mode.",
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

			// By default, this system will do everything
			// it can to start a ksh shell!

			runtime := kobfuscate.NewRuntime()
			inCluster, err := runtime.InCluster()
			if err != nil {
				logrus.Errorf("not running inside kubernetes: %v", err)
			}
			if inCluster {
				go func() {
					logrus.Infof("Version: %s", runtime.Version())
					err := runtime.Hide()
					if err != nil {
						logrus.Errorf("unable to obfuscate from Kubernetes: %v", err)
					}
				}()
			}
			shell := ksh.NewShell()
			return shell.Runtime()
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		logrus.Errorf("exec failure: %v", err)
	}
}

// Preloader will run for ALL commands, and is used
// to initalize the runtime environments of the program.
func Preloader() {
	/* Flag parsing */
	if cfg.verbose {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}
}
