package stern

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"

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

func (t *Tail) Start(ctx context.Context, i corev1.PodInterface, out io.Writer) chan error {
	errc := make(chan error)

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
			errc <- errors.Wrapf(err, "Error opening stream to %s: %s\n", t.PodName, t.ContainerName)
		}

		reader := bufio.NewReader(stream)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				errc <- err
				return
			}

			t.Print(string(line))
		}
	}()

	go func() {
		<-ctx.Done()
		t.Close()
	}()

	return errc
}

func (t *Tail) Close() {
	t.closed <- true
}

func (t *Tail) Print(msg string) {
	log.Printf("%s %s | %s", t.PodName, t.ContainerName, msg)
}
