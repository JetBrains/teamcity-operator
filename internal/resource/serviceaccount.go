package resource

import (
	"context"
	"fmt"
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
	return []client.Object{
		builder.getEmptyServiceAccount(),
	}, nil
}

func (builder ServiceAccountBuilder) Update(object client.Object) error {
	serviceAccount := object.(*v1.ServiceAccount)
	expected := &builder.Instance.Spec.ServiceAccount
	serviceAccount.Annotations = expected.Annotations
	if err := controllerutil.SetControllerReference(builder.Instance, serviceAccount, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder ServiceAccountBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	if !builder.Instance.ServiceAccountProvided() {
		return []client.Object{
			builder.getEmptyServiceAccount(),
		}, nil
	}
	return nil, nil
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
