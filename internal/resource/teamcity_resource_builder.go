package resource

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TeamCityResourceBuilder struct {
	Instance *TeamCity
	Scheme   *runtime.Scheme
	Client   client.Client
}

type ResourceBuilder interface {
	BuildObjectList() ([]client.Object, error)
	Update(object client.Object) error
	GetObsoleteObjects(ctx context.Context) ([]client.Object, error)
	UpdateMayRequireStsRecreate() bool
}

func (builder *TeamCityResourceBuilder) ResourceBuilders() []ResourceBuilder {

	builders := []ResourceBuilder{
		builder.StatefulSet(),
		builder.Service(),
		builder.Ingress(),
		builder.PersistentVolumeClaim(),
	}

	return builders
}
