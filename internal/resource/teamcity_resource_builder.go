package resource

import (
	"git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TeamCityResourceBuilder struct {
	Instance *v1beta1.TeamCity
	Scheme   *runtime.Scheme
}

type ResourceBuilder interface {
	Build() (client.Object, error)
	Update(object client.Object) error
	UpdateMayRequireStsRecreate() bool
}

func (builder *TeamCityResourceBuilder) ResourceBuilders() []ResourceBuilder {

	builders := []ResourceBuilder{
		builder.StatefulSet(),
	}
	return builders
}
