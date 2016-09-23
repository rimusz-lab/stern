package stern

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/wercker/stern/kubernetes"
)

func Run(ctx context.Context, config *Config) error {
	clientset, err := kubernetes.NewClientSet(config.KubeConfig, config.ContextName)
	if err != nil {
		return err
	}

	input := clientset.Core().Pods(config.Namespace)

	added, removed, err := Watch(ctx, input, config.PodQuery)
	if err != nil {
		return errors.Wrap(err, "failed to set up watch")
	}

	tails := make(map[string]*Tail)

	go func() {
		for p := range added {
			ID := id(p.Pod, p.Container)
			if tails[ID] != nil {
				continue
			}

			fmt.Printf("addded %s | %s\n", p.Pod, p.Container)

			tail := NewTail(p.Pod, p.Container, &TailOptions{
				Timestamps:   config.Timestamps,
				SinceSeconds: config.Since,
			})
			tails[ID] = tail

			tail.Start(ctx, input, nil)
		}
	}()

	go func() {
		for p := range removed {
			ID := id(p.Pod, p.Container)
			if tails[ID] == nil {
				continue
			}
			fmt.Printf("removed %s | %s\n", p.Pod, p.Container)
			tails[ID].Close()
			delete(tails, ID)
		}
	}()

	<-ctx.Done()

	return nil
}

func id(podID string, containerID string) string {
	return fmt.Sprintf("%s-%s", podID, containerID)
}
