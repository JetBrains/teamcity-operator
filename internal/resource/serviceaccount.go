package resource

import (
	"context"
	"fmt"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ServiceAccountBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) ServiceAccount() *ServiceAccountBuilder {
	return &ServiceAccountBuilder{builder}
}

func (builder ServiceAccountBuilder) BuildObjectList() ([]client.Object, error) {
	objectList := []client.Object{}
	if builder.Instance.ServiceAccountProvided() {
		objectList = append(objectList, builder.getEmptyServiceAccount())
	}
	return objectList, nil
}

func (builder ServiceAccountBuilder) Update(object client.Object) error {
	serviceAccount := object.(*v1.ServiceAccount)
	expected := &builder.Instance.Spec.ServiceAccount
	serviceAccount.Annotations = expected.Annotations
	serviceAccount.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)
	if err := controllerutil.SetControllerReference(builder.Instance, serviceAccount, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder ServiceAccountBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	currentServiceAccountList := &v1.ServiceAccountList{}
	obsoleteObjects := []client.Object{}
	listOptions := []client.ListOption{
		client.InNamespace(builder.Instance.Namespace),
		client.MatchingLabels(metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)),
	}
	if err := builder.Client.List(ctx, currentServiceAccountList, listOptions...); err != nil {
		return nil, err
	}
	for _, serviceAccount := range currentServiceAccountList.Items {
		sa := serviceAccount
		if !builder.Instance.ServiceAccountProvided() || sa.Name != builder.Instance.Spec.ServiceAccount.Name {
			obsoleteObjects = append(obsoleteObjects, &sa)
		}
	}
	return obsoleteObjects, nil
}

func (builder ServiceAccountBuilder) UpdateMayRequireStsRecreate() bool {
	return false
}

func (builder ServiceAccountBuilder) getEmptyServiceAccount() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Instance.Spec.ServiceAccount.Name,
			Namespace: builder.Instance.Namespace,
		},
	}
}
