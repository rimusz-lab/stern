package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"sync"

	"github.com/fatih/color"
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
	app.Action = tailAction

	app.Run(os.Args)
}

type Config struct {
	KubeConfig     string
	PodQuery       *regexp.Regexp
	Timestamps     bool
	ContainerQuery *regexp.Regexp
	Since          int64
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
		fmt.Println(err)
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
		Timestamps:     c.Bool("timestamps"),
		Since:          c.Int64("since"),
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

	fmt.Println("Getting pods..")
	res, err := pods.List(api.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list pods")
	}

	var wg sync.WaitGroup

	colorList := [][2]*color.Color{
		{color.New(color.FgHiCyan), color.New(color.FgCyan)},
		{color.New(color.FgHiGreen), color.New(color.FgGreen)},
		{color.New(color.FgHiMagenta), color.New(color.FgMagenta)},
		{color.New(color.FgHiYellow), color.New(color.FgYellow)},
		{color.New(color.FgHiBlue), color.New(color.FgBlue)},
		{color.New(color.FgHiRed), color.New(color.FgRed)},
	}

	counter := 0
	for _, pod := range res.Items {
		if config.PodQuery.MatchString(pod.Name) {
			index := counter
			counter++
			pod := pod

			colorIndex := index % len(colorList)
			podLog := colorList[colorIndex][0]
			containerLog := colorList[colorIndex][1]
			podLog.Println(pod.Name)

			// check containers
			for _, container := range pod.Spec.Containers {
				wg.Add(1)
				go func() {
					defer wg.Done()

					req := clientset.Core().Pods("default").GetLogs(pod.Name, &v1api.PodLogOptions{
						Follow:       true,
						Timestamps:   config.Timestamps,
						Container:    container.Name,
						SinceSeconds: &config.Since,
					})

					readCloser, err := req.Stream()
					if err != nil {
						fmt.Println(errors.Wrap(err, "could not open stream"))
					}
					defer readCloser.Close()

					stream, err := req.Stream()
					if err != nil {
						log.Printf("Error opening stream to %s: %s\n", pod.Name, err.Error())
						continue
					}

					reader := bufio.NewReader(stream)

					for {
						podLog.Printf("%-32s ", pod.Name)
						if container.Name != "" {
							containerLog.Printf("%-12s ", container.Name)
						}

						line, err := reader.ReadBytes('\n')
						if err != nil {
							// EOF -> pod exited
							if err == io.EOF {
								color.Red("terminated")
								return
							}

							fmt.Println(err)
							return
						}

						fmt.Printf("%s", line)
					}
				}()
			}
		}
	}

	if counter == 0 {
		fmt.Println("No matches")
	}

	wg.Wait()

	// monitor for pods added/removed
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
