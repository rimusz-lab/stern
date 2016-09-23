package stern

import (
	"context"
	"regexp"

	"github.com/pkg/errors"

	corev1 "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/watch"
)

type Target struct {
	Pod       string
	Container string
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
				pod := e.Object.(*v1.Pod)

				if !podFilter.MatchString(pod.Name) {
					continue
				}

				switch e.Type {
				case watch.Added:
					for _, c := range pod.Status.ContainerStatuses {
						if c.Ready {
							added <- &Target{
								Pod:       pod.Name,
								Container: c.Name,
							}
						}
					}
				case watch.Modified:
					for _, c := range pod.Status.ContainerStatuses {
						if c.Ready {
							added <- &Target{
								Pod:       pod.Name,
								Container: c.Name,
							}
						}
					}
				case watch.Deleted:
					for _, container := range pod.Spec.Containers {
						removed <- &Target{
							Pod:       pod.Name,
							Container: container.Name,
						}
					}
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
