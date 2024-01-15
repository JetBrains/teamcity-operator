package resource

import (
	"context"
	"errors"
	"fmt"
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type IngressBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) Ingress() *IngressBuilder {
	return &IngressBuilder{builder}
}

func (builder *IngressBuilder) UpdateMayRequireStsRecreate() bool {
	return true
}

func (builder *IngressBuilder) BuildObjectList() ([]client.Object, error) {
	var objectList []client.Object
	for _, ingress := range builder.Instance.Spec.IngressList {
		objectList = append(objectList, &netv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingress.Name, Namespace: builder.Instance.Namespace},
		})
	}
	return objectList, nil
}

func (builder *IngressBuilder) Update(object client.Object) error {
	var idx int
	ingressList := builder.Instance.Spec.IngressList
	if idx = getIngressIndex(object, ingressList); idx == -1 {
		return fmt.Errorf("failed to update object: %w", errors.New("the specified Ingress does not exist: "+object.GetName()))
	}
	desired := ingressList[idx]
	current := object.(*netv1.Ingress)
	current.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)
	current.Spec = desired.IngressSpec
	current.Annotations = desired.Annotations
	if err := controllerutil.SetControllerReference(builder.Instance, current, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder *IngressBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	currentIngressList := &netv1.IngressList{}
	obsoleteObjects := []client.Object{}
	listOtions := []client.ListOption{
		client.InNamespace(builder.Instance.Namespace),
		client.MatchingLabels(metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)),
	}
	if err := builder.Client.List(ctx, currentIngressList, listOtions...); err != nil {
		return nil, err
	}
	for _, ingress := range currentIngressList.Items {
		var idx int
		ing := ingress
		if idx = getIngressIndex(&ingress, builder.Instance.Spec.IngressList); idx == -1 {
			obsoleteObjects = append(obsoleteObjects, &ing)
		}
	}
	return obsoleteObjects, nil
}

func getIngressIndex(object client.Object, ingressList []v1alpha1.Ingress) int {
	for idx, ingress := range ingressList {
		if ingress.Name == object.GetName() {
			return idx
		}
	}
	return -1
}
