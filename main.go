package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"

	cli "gopkg.in/urfave/cli.v1"

	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/tools/clientcmd"

	v1api "k8s.io/client-go/1.4/pkg/api/v1"
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

	ctx, cancel := context.WithCancel(context.Background())

	err = run(ctx, config)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}

	go func() {
		time.Sleep(time.Second)
		cancel()
	}()

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

	c.Insecure = true

	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	var pods v1.PodInterface // this fixes autocomplete
	pods = clientset.Core().Pods("")

	log.Println("Getting pods..")
	res, err := pods.List(api.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list pods")
	}

	var wg sync.WaitGroup

	match := false
	for _, pod := range res.Items {
		if config.PodQuery.MatchString(pod.Name) {
			log.Printf("tailing %s", pod.Name)
			wg.Add(1)
			match = true
			pod := pod

			go func() {
				req := clientset.Core().Pods("default").GetLogs(pod.Name, &v1api.PodLogOptions{
					Follow:     true,
					Timestamps: true,
					Container:  "",
				})

				readCloser, err := req.Stream()
				if err != nil {
					log.Panic(errors.Wrap(err, "could not open stream"))
				}
				defer readCloser.Close()

				stream, err := req.Stream()
				if err != nil {
					log.Printf("Error opening stream to %s: %s\n", pod.Name, err.Error())
				}

				reader := bufio.NewReader(stream)

				for {
					line, err := reader.ReadBytes('\n')
					if err != nil {
						log.Printf("ERROR: %s\n", err.Error())
					}

					fmt.Printf("%32s | %s", pod.Name, line)
				}
			}()
		}
	}

	if !match {
		log.Println("No matches")
	}

	wg.Wait()

	// watch, err := pods.Watch(api.ListOptions{})

	// go func() {
	// 	for {
	// 		select {
	// 		case e := <-watch.ResultChan():
	// 			log.Println("EVENT", e)
	// 		case <-ctx.Done():
	// 			watch.Stop()
	// 		}
	// 	}
	// }()

	return nil
}
