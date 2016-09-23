package stern

import (
	"context"
	"fmt"

	"github.com/wercker/stern/kubernetes"
)

func Run(ctx context.Context, config *Config) error {
	clientset, err := kubernetes.NewClientSet(config.KubeConfig, config.ContextName)
	if err != nil {
		return err
	}

	fmt.Println(clientset)

	return nil
}
