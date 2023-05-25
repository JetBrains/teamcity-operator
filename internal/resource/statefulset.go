package resource

import (
	"fmt"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type StatefulSetBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) StatefulSet() *StatefulSetBuilder {
	return &StatefulSetBuilder{builder}
}

func (builder *StatefulSetBuilder) UpdateMayRequireStsRecreate() bool {
	return true
}

func (builder *StatefulSetBuilder) Build() (client.Object, error) {
	return &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Instance.Name,
			Namespace: builder.Instance.Namespace,
		},
		Spec: v1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: metadata.LabelSelector(builder.Instance.Name),
			},
		},
	}, nil
}

func (builder *StatefulSetBuilder) Update(object client.Object) error {
	statefulSet := object.(*v1.StatefulSet)

	statefulSet.Spec.Replicas = builder.Instance.Spec.Replicas
	statefulSet.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)

	statefulSet.Spec.Template.Labels = metadata.Label(builder.Instance.Name)
	statefulSet.Spec.Template.Spec.Containers = []v12.Container{}
	statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, v12.Container{})
	statefulSet.Spec.Template.Spec.Containers[0].Image = builder.Instance.Spec.Image

	if err := controllerutil.SetControllerReference(builder.Instance, statefulSet, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}

	return nil
}
