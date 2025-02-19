package checkpoint

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildCheckpoint(instanceName string, instanceNamespace string, stage Stage) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConstructCheckpointName(instanceName),
			Namespace: instanceNamespace,
		},
		Data: map[string]string{
			"stage": stage.String(),
		},
	}
}

func ConstructCheckpointName(instanceName string) string {
	return fmt.Sprintf("update-checkpoint-%s", instanceName)
}
