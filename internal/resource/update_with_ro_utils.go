package resource

import (
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	RoNodeRole             = "update-with-ro"
	RoNodePostfix          = "-update-replica"
	UpdatePolicyAnnotation = "update-policy"
	UpdateWithRoPolicy     = "update-with-ro"
)

func BuildRoNode(instance *TeamCity, name string) Node {
	return Node{
		Name: name,
		Spec: NodeSpec{
			Requests: instance.Spec.MainNode.Spec.Requests,
		},
	}
}

func GetROStatefulSetNamespacedName(instance *TeamCity) types.NamespacedName {
	return types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Spec.MainNode.Name + RoNodePostfix,
	}
}

func BuildROStatefulSet(instance *TeamCity) *v1.StatefulSet {
	node := BuildRoNode(instance, GetROStatefulSetNamespacedName(instance).Name)
	labels := metadata.GetStatefulSetLabels(instance.Name, node.Name, RoNodeRole, instance.Labels)
	roStatefulset := v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
	}
	return &roStatefulset
}

func UpdateROStatefulSet(scheme *runtime.Scheme, instance *TeamCity,
	mainStatefulSet *v1.StatefulSet, roStatefulSet *v1.StatefulSet) error {

	node := BuildRoNode(instance, GetROStatefulSetNamespacedName(instance).Name)
	labels := metadata.GetStatefulSetLabels(instance.Name, node.Name, RoNodeRole, instance.Labels)
	roStatefulSet.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
	roStatefulSet.Spec.Template.Spec = mainStatefulSet.Spec.Template.Spec
	roStatefulSet.Spec.Template.Labels = labels
	envVars := BuildEnvVariablesFromGlobalAndNodeSpecificSettings(instance, node)
	roStatefulSet.Spec.Template.Spec.Containers[0].Env = envVars
	if err := controllerutil.SetControllerReference(instance, roStatefulSet, scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func ChangesRequireNodeStatefulSetRestart(instance *TeamCity, node Node, existing *v1.StatefulSet) bool {
	var desired v1.StatefulSet
	ConfigureStatefulSet(instance, node, &desired)
	var container v12.Container
	ConfigureContainer(instance, node, &container)
	desired.Spec.Template.Spec.Containers = []v12.Container{container}

	if !equality.Semantic.DeepDerivative(desired.Spec, existing.Spec) {
		return true
	}
	return false
}

func UpdateWithRo(node Node) bool {
	return node.Annotations[UpdatePolicyAnnotation] == UpdateWithRoPolicy
}
