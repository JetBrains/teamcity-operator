package resource

import (
	"context"
	"errors"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SecondaryStatefulSetBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) SecondaryStatefulSet() *SecondaryStatefulSetBuilder {
	return &SecondaryStatefulSetBuilder{builder}
}

func (builder SecondaryStatefulSetBuilder) BuildObjectList() ([]client.Object, error) {
	var objectList []client.Object
	for _, secondaryNode := range builder.Instance.Spec.SecondaryNodes {
		nodeLabels := metadata.GetStatefulSetLabels(builder.Instance.Name, secondaryNode.Name, "secondary", builder.Instance.Labels)
		node := CreateEmptyStatefulSet(secondaryNode.Name, builder.Instance.Namespace, nodeLabels)
		objectList = append(objectList, &node)
	}
	return objectList, nil
}

func (builder SecondaryStatefulSetBuilder) Update(object client.Object) error {
	var idx int
	secondaryNodeList := builder.Instance.Spec.SecondaryNodes
	if idx = builder.getNodeIndex(object, secondaryNodeList); idx == -1 {
		return fmt.Errorf("failed to update object: %w", errors.New("the specified Statefulset does not exist: "+object.GetName()))
	}
	desired := secondaryNodeList[idx]

	statefulSpec := object.(*v1.StatefulSet)

	ConfigureStatefulSet(builder.Instance, desired, statefulSpec)
	var container v12.Container
	ConfigureContainer(builder.Instance, desired, &container)

	statefulSpec.Spec.Template.Spec.Containers = []v12.Container{container}

	if err := controllerutil.SetControllerReference(builder.Instance, statefulSpec, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder SecondaryStatefulSetBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	currentStatefulList := &v1.StatefulSetList{}
	obsoleteObjects := []client.Object{}
	secondaryNodeLabels := metadata.GetStatefulSetCommonLabels(builder.Instance.Name, "secondary", builder.Instance.Labels)
	listOptions := []client.ListOption{
		client.InNamespace(builder.Instance.Namespace),
		client.MatchingLabels(secondaryNodeLabels),
	}
	if err := builder.Client.List(ctx, currentStatefulList, listOptions...); err != nil {
		return nil, err
	}
	for _, statefulSet := range currentStatefulList.Items {
		var idx int
		sts := statefulSet
		if idx = builder.getNodeIndex(&statefulSet, builder.Instance.Spec.SecondaryNodes); idx == -1 {
			obsoleteObjects = append(obsoleteObjects, &sts)
		}
	}
	return obsoleteObjects, nil
}

func (builder SecondaryStatefulSetBuilder) UpdateMayRequireStsRecreate() bool {
	return true
}

func (builder SecondaryStatefulSetBuilder) getNodeIndex(object client.Object, nodeList []Node) int {
	for idx, node := range nodeList {
		if node.Name == object.GetName() {
			return idx
		}
	}
	return -1
}
