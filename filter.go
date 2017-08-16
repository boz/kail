package kail

import "k8s.io/api/core/v1"

type ContainerFilter interface {
	Accept(cs v1.ContainerStatus) bool
}

func NewContainerFilter(names []string) ContainerFilter {
	return containerFilter(names)
}

type containerFilter []string

func (cf containerFilter) Accept(cs v1.ContainerStatus) bool {
	if !cs.Ready {
		return false
	}
	if len(cf) == 0 {
		return true
	}
	for _, name := range cf {
		if name == cs.Name {
			return true
		}
	}
	return false
}
