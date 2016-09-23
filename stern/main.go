package stern

import (
	"context"
	"log"

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

	go func() {
		for p := range added {
			log.Println("add", p.Pod.Name, p.Container.Name)
		}
	}()

	go func() {
		for p := range removed {
			log.Println("remove", p.Pod.Name, p.Container.Name)
		}
	}()

	<-ctx.Done()

	return nil
}
