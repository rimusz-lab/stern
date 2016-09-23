package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"

	"github.com/pkg/errors"
	"github.com/wercker/stern/stern"

	cli "gopkg.in/urfave/cli.v1"
)

func Run() {
	app := cli.NewApp()

	app.Name = "stern"
	app.Usage = "Tail multiple pods and containers from Kubernetes"
	app.UsageText = "stern [options] query"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "kube-config",
			Value: "",
		},
		cli.StringFlag{
			Name:  "container, c",
			Usage: "container name when multiple containers in pod",
			Value: ".*",
		},
		cli.StringFlag{
			Name:  "context",
			Usage: "kubernetes context to use",
			Value: "",
		},
		cli.StringFlag{
			Name:  "namespace",
			Usage: "kubernetes namespace to use",
			Value: "default",
		},
		cli.BoolFlag{
			Name:  "timestamps, t",
			Usage: "print timestamps",
		},
		cli.Int64Flag{
			Name:  "since, s",
			Usage: "since X seconds ago",
			Value: 10,
		},
	}

	app.Action = func(c *cli.Context) error {
		if len(c.Args()) != 1 {
			return cli.ShowAppHelp(c)
		}

		config, err := parseConfig(c)
		if err != nil {
			log.Println(err)
			os.Exit(2)
		}

		ctx := context.Background()

		err = stern.Run(ctx, config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return nil
	}

	app.Run(os.Args)
}

func parseConfig(c *cli.Context) (*stern.Config, error) {
	kubeConfig := c.String("kube-config")
	if kubeConfig == "" {
		// kubernetes requires an absolute path
		u, err := user.Current()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current user")
		}

		kubeConfig = path.Join(u.HomeDir, ".kube/config")
	}

	pod, err := regexp.Compile(c.Args()[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	container, err := regexp.Compile(c.String("container"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	return &stern.Config{
		KubeConfig:     kubeConfig,
		PodQuery:       pod,
		ContainerQuery: container,
		Timestamps:     c.Bool("timestamps"),
		Since:          c.Int64("since"),
		ContextName:    c.String("context"),
		Namespace:      c.String("namespace"),
	}, nil
}
