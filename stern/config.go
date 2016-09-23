package stern

import "regexp"

type Config struct {
	KubeConfig     string
	ContextName    string
	Namespace      string
	PodQuery       *regexp.Regexp
	Timestamps     bool
	ContainerQuery *regexp.Regexp
	Since          int64
}
