package stern

import (
	"context"
	"log"
	"regexp"

	"github.com/pkg/errors"

	corev1 "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/watch"
)

type Target struct {
	Pod       *v1.Pod
	Container *v1.Container
}

func Watch(ctx context.Context, i corev1.PodInterface, podFilter *regexp.Regexp) (chan *Target, chan *Target, error) {
	watcher, err := i.Watch(api.ListOptions{Watch: true})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to set up watch")
	}

	added := make(chan *Target)
	removed := make(chan *Target)

	go func() {
		for {
			select {
			case e := <-watcher.ResultChan():
				pod, ok := e.Object.(*v1.Pod)
				if !ok {
					continue
				}

				if !podFilter.MatchString(pod.Name) {
					continue
				}

				switch e.Type {
				case watch.Added:
					log.Println("added", pod.Name)
				// case watch.Deleted:
				// 	added <- pod
				case watch.Modified:
					log.Println("modified", pod.Name, pod.Status.Reason, pod.Status.ContainerStatuses)
				}
			case <-ctx.Done():
				watcher.Stop()
				close(added)
				close(removed)
			}
		}
	}()

	return added, removed, nil
}
