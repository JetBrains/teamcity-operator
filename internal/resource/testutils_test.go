package resource

import (
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	TeamCityName      = "test"
	TeamCityNamespace = "default"
	TeamCityImage     = "jetbrains/teamcity-server:latest"
)

type ResourceModifier func(*v1alpha1.TeamCity)

var (
	Instance                  v1alpha1.TeamCity
	DefaultStatefulSetBuilder *StatefulSetBuilder

	scheme           *runtime.Scheme
	builder          *TeamCityResourceBuilder
	teamCityReplicas = int32(0)

	pvcAccessMode       = []corev1.PersistentVolumeAccessMode{"ReadWriteMany"}
	pvcStorageClassName = "standard"
	pvcVolumeMode       = corev1.PersistentVolumeFilesystem
	pvcResources        = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("1Gi"),
		},
	}
	pvcName = "test-pvc"
	pvcSpec = corev1.PersistentVolumeClaimSpec{
		AccessModes:      pvcAccessMode,
		StorageClassName: &pvcStorageClassName,
		VolumeMode:       &pvcVolumeMode,
		Resources:        pvcResources,
	}
	pvc = v1alpha1.CustomPersistentVolumeClaim{
		Name: pvcName,
		Spec: pvcSpec,
		VolumeMount: corev1.VolumeMount{
			Name:      "default-storage",
			MountPath: "/storage",
		},
	}
	requests = corev1.ResourceList{
		"cpu":    resource.MustParse("750m"),
		"memory": resource.MustParse("1000Mi"),
	}
	xmxPercentage  = int64(95)
	databaseSecret = v1alpha1.DatabaseSecret{
		Secret: "database-secret",
	}
)

func BeforeEachBuild(modify ResourceModifier) {
	Instance = GetBaseTcInstance()
	scheme = runtime.NewScheme()

	Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())

	modify(&Instance)

	builder = &TeamCityResourceBuilder{
		Instance: &Instance,
		Scheme:   scheme,
	}
	DefaultStatefulSetBuilder = builder.StatefulSet()
}

func BeforeEachUpdate(modify ResourceModifier) {
	Instance = GetBaseTcInstance()
	scheme = runtime.NewScheme()

	Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())

	modify(&Instance)

	builder = &TeamCityResourceBuilder{
		Instance: &Instance,
		Scheme:   scheme,
	}
	DefaultStatefulSetBuilder = builder.StatefulSet()
}

func GetBaseTcInstance() v1alpha1.TeamCity {
	return v1alpha1.TeamCity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TeamCityName,
			Namespace: TeamCityNamespace,
		},
		Spec: v1alpha1.TeamCitySpec{
			Image:                  TeamCityImage,
			Replicas:               &teamCityReplicas,
			PersistentVolumeClaims: []v1alpha1.CustomPersistentVolumeClaim{pvc},
			Requests:               requests,
			XmxPercentage:          xmxPercentage,
			DatabaseSecret:         databaseSecret,
		},
	}
}
