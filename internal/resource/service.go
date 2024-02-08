package resource

import (
	"context"
	"errors"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ServiceBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) Service() *ServiceBuilder {
	return &ServiceBuilder{builder}
}

func (builder *ServiceBuilder) UpdateMayRequireStsRecreate() bool {
	return true
}

func (builder *ServiceBuilder) BuildObjectList() ([]client.Object, error) {
	var objectList []client.Object
	for _, service := range builder.Instance.Spec.ServiceList {
		objectList = append(objectList, &v12.Service{
			ObjectMeta: metav1.ObjectMeta{Name: service.Name, Namespace: builder.Instance.Namespace},
		})
	}
	return objectList, nil
}

func (builder *ServiceBuilder) Update(object client.Object) error {
	var idx int
	serviceList := builder.Instance.Spec.ServiceList
	if idx = getServiceIndex(object, serviceList); idx == -1 {
		return fmt.Errorf("failed to update object: %w", errors.New("the specified Service does not exist: "+object.GetName()))
	}
	desired := serviceList[idx]
	current := object.(*v12.Service)
	current.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)
	current.Annotations = desired.Annotations
	current.Spec = desired.ServiceSpec
	if err := controllerutil.SetControllerReference(builder.Instance, current, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}

	return nil
}

func (builder *ServiceBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	currentServiceList := &v12.ServiceList{}
	obsoleteObjects := []client.Object{}
	listOptions := []client.ListOption{
		client.InNamespace(builder.Instance.Namespace),
		client.MatchingLabels(metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)),
	}
	if err := builder.Client.List(ctx, currentServiceList, listOptions...); err != nil {
		return nil, err
	}
	for _, service := range currentServiceList.Items {
		var idx int
		s := service
		if idx = getServiceIndex(&service, builder.Instance.Spec.ServiceList); idx == -1 {
			obsoleteObjects = append(obsoleteObjects, &s)
		}
	}
	return obsoleteObjects, nil
}

func getServiceIndex(object client.Object, serviceList []Service) int {
	for idx, service := range serviceList {
		if service.Name == object.GetName() {
			return idx
		}
	}
	return -1
}
