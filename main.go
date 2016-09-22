package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"time"

	"github.com/pkg/errors"

	cli "gopkg.in/urfave/cli.v1"

	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/tools/clientcmd"
)

func main() {
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
	}
	app.Action = tailAction

	app.Run(os.Args)
}

type Config struct {
	KubeConfig     string
	PodQuery       *regexp.Regexp
	ContainerQuery *regexp.Regexp
}

var tailAction = func(c *cli.Context) error {
	config, err := parseConfig(c)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()

	err = run(ctx, config)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}

	return nil
}

func parseConfig(c *cli.Context) (*Config, error) {
	kubeConfig := c.String("kube-config")
	if kubeConfig == "" {
		// kubernetes requires an absolute path
		u, err := user.Current()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current user")
		}

		kubeConfig = path.Join(u.HomeDir, ".kube/config")
	}

	if len(c.Args()) < 1 {
		return nil, errors.New("query missing")
	}

	pod, err := regexp.Compile(c.Args()[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	container, err := regexp.Compile(c.String("container"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	return &Config{
		KubeConfig:     kubeConfig,
		PodQuery:       pod,
		ContainerQuery: container,
	}, nil
}

func run(ctx context.Context, config *Config) error {
	c, err := clientcmd.BuildConfigFromFlags("", config.KubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to get kube config")
	}

	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	for {
		pods, err := clientset.Core().Pods("cluster-manager").List(api.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get pods")
		}

		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		time.Sleep(10 * time.Second)
	}

	return nil
}
