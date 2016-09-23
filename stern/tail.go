package stern

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"

	corev1 "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/rest"
)

type Tail struct {
	PodName       string
	ContainerName string
	Options       *TailOptions
	req           *rest.Request
	closed        chan bool
}

type TailOptions struct {
	Timestamps   bool
	SinceSeconds int64
}

func NewTail(podName, containerName string, options *TailOptions) *Tail {
	return &Tail{PodName: podName, ContainerName: containerName, Options: options}
}

func (t *Tail) Start(ctx context.Context, i corev1.PodInterface, out io.Writer) {
	go func() {
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

		fmt.Println("listening to", t.PodName, t.ContainerName)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					t.Print("EOF\n")
				}
				return
			}

			t.Print(string(line))
		}
	}()

	go func() {
		<-ctx.Done()
		t.Close()
	}()
}

func (t *Tail) Close() {
	fmt.Printf("close %s %s\n", t.PodName, t.ContainerName)
	t.closed <- true
}

func (t *Tail) Print(msg string) {
	fmt.Printf("MSG %s %s | %s", t.PodName, t.ContainerName, msg)
}
