package handler

import (
	"context"

	"lcp.io/lcp/lib/runtime"
)

type ObjectMeta struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type Pod struct {
	runtime.TypeMeta `json:",inline" yaml:",inline"`
	Metadata         ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec             PodSpec    `json:"spec" yaml:"spec"`
}

func (p *Pod) GetTypeMeta() *runtime.TypeMeta { return &p.TypeMeta }

func (p *Pod) Get(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
	return p, nil
}

func NewPod() *Pod {
	return &Pod{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		Metadata: ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
			Labels:    map[string]string{"app": "nginx"},
		},
		Spec: PodSpec{
			Containers: []Container{
				{Name: "nginx", Image: "nginx:1.25"},
			},
		},
	}
}

type PodSpec struct {
	Containers []Container `json:"containers" yaml:"containers"`
}

type Container struct {
	Name  string `json:"name" yaml:"name"`
	Image string `json:"image" yaml:"image"`
}
