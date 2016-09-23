package stern

import (
	"bufio"
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/pkg/errors"

	corev1 "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/rest"
)

type Tail struct {
	PodName        string
	ContainerName  string
	Options        *TailOptions
	req            *rest.Request
	closed         chan bool
	podColor       *color.Color
	containerColor *color.Color
}

type TailOptions struct {
	Timestamps   bool
	SinceSeconds int64
}

func NewTail(podName, containerName string, options *TailOptions) *Tail {
	return &Tail{
		PodName:       podName,
		ContainerName: containerName,
		Options:       options,
		closed:        make(chan bool),
	}
}

var index = 0

var colorList = [][2]*color.Color{
	{color.New(color.FgHiCyan), color.New(color.FgCyan)},
	{color.New(color.FgHiGreen), color.New(color.FgGreen)},
	{color.New(color.FgHiMagenta), color.New(color.FgMagenta)},
	{color.New(color.FgHiYellow), color.New(color.FgYellow)},
	{color.New(color.FgHiBlue), color.New(color.FgBlue)},
	{color.New(color.FgHiRed), color.New(color.FgRed)},
}

func (t *Tail) Start(ctx context.Context, i corev1.PodInterface) {
	index++

	colorIndex := index % len(colorList)
	t.podColor = colorList[colorIndex][0]
	t.containerColor = colorList[colorIndex][1]

	go func() {
		g := color.New(color.FgGreen, color.Bold).SprintFunc()
		fmt.Printf("%s %s %s\n", g("+"), t.PodName, t.ContainerName)

		req := i.GetLogs(t.PodName, &v1.PodLogOptions{
			Follow:       true,
			Timestamps:   t.Options.Timestamps,
			Container:    t.ContainerName,
			SinceSeconds: &t.Options.SinceSeconds,
		})

		readCloser, err := req.Stream()
		if err != nil {
			fmt.Println(errors.Wrap(err, "could not open stream"))
		}
		defer readCloser.Close()

		stream, err := req.Stream()
		if err != nil {
			fmt.Println(errors.Wrapf(err, "Error opening stream to %s: %s\n", t.PodName, t.ContainerName))
		}

		reader := bufio.NewReader(stream)

		for {
			select {
			case <-t.closed:
				return
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					continue
				}

				t.Print(string(line))
			}
		}
	}()

	go func() {
		<-ctx.Done()
		t.closed <- true
	}()
}

func (t *Tail) Close() {
	r := color.New(color.FgRed, color.Bold).SprintFunc()
	fmt.Printf("%s %s %s\n", r("-"), t.PodName, t.ContainerName)
	t.closed <- true
}

func (t *Tail) Print(msg string) {
	p := t.podColor.SprintFunc()
	c := t.podColor.SprintFunc()
	fmt.Printf("%s %s %s", p(t.PodName), c(t.ContainerName), msg)
}
