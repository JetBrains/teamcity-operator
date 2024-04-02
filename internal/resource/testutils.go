package resource

import (
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TeamCityName      = "test"
	TeamCityNamespace = "default"
	TeamCityImage     = "jetbrains/teamcity-server:latest"
)

type ResourceModifier func(*TeamCity)

var (
	Instance                            TeamCity
	DefaultClient                       client.Client
	DefaultStatefulSetBuilder           *StatefulSetBuilder
	DefaultServiceBuilder               *ServiceBuilder
	DefaultIngressBuilder               *IngressBuilder
	DefaultPersistentVolumeClaimBuilder *PersistentVolumeClaimBuilder
	DefaultSecondaryStatefulSetBuilder  *SecondaryStatefulSetBuilder
	DefaultServiceAccountBuilder        *ServiceAccountBuilder

	StaleServiceName        = "StaleService"
	StalePvcName            = "StalePvc"
	StaleIngressName        = "StaleIngress"
	StaleServiceAccountName = "StaleServiceAccount"

	scheme           *runtime.Scheme
	builder          *TeamCityResourceBuilder
	teamCityReplicas = int32(0)

	dataDirPVCAccessMode       = []corev1.PersistentVolumeAccessMode{"ReadWriteMany"}
	dataDirPVCStorageClassName = "standard"
	dataDirPVCVolumeMode       = corev1.PersistentVolumeFilesystem
	dataDirPVCResources        = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("1Gi"),
		},
	}
	dataDirPVCName = "data-dir"
	dataDirPVCSpec = corev1.PersistentVolumeClaimSpec{
		AccessModes:      dataDirPVCAccessMode,
		StorageClassName: &dataDirPVCStorageClassName,
		VolumeMode:       &dataDirPVCVolumeMode,
		Resources:        dataDirPVCResources,
	}
	dataDirPVC = CustomPersistentVolumeClaim{
		Name: dataDirPVCName,
		Spec: dataDirPVCSpec,
		VolumeMount: corev1.VolumeMount{
			Name:      "default-storage",
			MountPath: "/storage",
		},
	}
	requests = corev1.ResourceList{
		"cpu":    resource.MustParse("750m"),
		"memory": resource.MustParse("1000Mi"),
	}
	mainNodeName         = "main-node"
	xmxPercentage        = int64(95)
	serviceNameMain      = TeamCityName + "-service-main"
	serviceNameSecondary = TeamCityName + "-service-secondary"
	servicePort          = int32(8111)

	ingressNameFirst     = TeamCityName + "-ingress-first"
	ingressNameSecondary = TeamCityName + "-ingress-secondary"
)

func BeforeEachBuild(modify ResourceModifier) {
	Instance = getBaseTcInstance()
	scheme = runtime.NewScheme()

	Expect(AddToScheme(scheme)).To(Succeed())
	Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())

	modify(&Instance)

	builder = &TeamCityResourceBuilder{
		Instance: &Instance,
		Scheme:   scheme,
		Client:   DefaultClient,
	}
	DefaultStatefulSetBuilder = builder.StatefulSet()
	DefaultServiceBuilder = builder.Service()
	DefaultIngressBuilder = builder.Ingress()
	DefaultPersistentVolumeClaimBuilder = builder.PersistentVolumeClaim()
	DefaultSecondaryStatefulSetBuilder = builder.SecondaryStatefulSet()
	DefaultServiceAccountBuilder = builder.ServiceAccount()
}

func getBaseTcInstance() TeamCity {
	return TeamCity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TeamCityName,
			Namespace: TeamCityNamespace,
		},
		Spec: TeamCitySpec{
			Image:              TeamCityImage,
			DataDirVolumeClaim: dataDirPVC,
			XmxPercentage:      xmxPercentage,
			MainNode: Node{
				Name: mainNodeName,
				Spec: NodeSpec{
					Requests: requests,
				},
			},
		},
	}
}

func getStartupConfigurations() map[string]string {
	return map[string]string{
		"foo":   "bar",
		"hello": "world",
	}
}

func getDatabaseSecret() DatabaseSecret {
	return DatabaseSecret{
		Secret: "database-secret",
	}
}

func getNodeSelector() map[string]string {
	return map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
}

func getAffinity() corev1.Affinity {
	return corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "some-key",
								Operator: "In",
								Values:   []string{"some-value"},
							},
						},
					},
				},
			},
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
				{
					Weight: 5,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "some-key",
								Operator: "In",
								Values:   []string{"some-value"},
							},
						},
					},
				},
			},
		},
	}
}

func getInitContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:  "volume-permissions",
			Image: "busybox",
			Command: []string{
				"sh",
				"-c",
				"chown -R 1000:1000 /storage",
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: pointer.Bool(false),
				RunAsUser:    pointer.Int64(0),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "teamcity-node-volume1",
					MountPath: "/storage",
				},
			},
		},
	}
}

func getAdditionalPVC() CustomPersistentVolumeClaim {
	return CustomPersistentVolumeClaim{
		Name: "some-additional-data",
		VolumeMount: corev1.VolumeMount{
			Name:      "plugin-data",
			MountPath: "/storage/plugins",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteMany"},
			Resources:        corev1.ResourceRequirements{},
			StorageClassName: pointer.String("standard"),
			VolumeMode:       &dataDirPVCVolumeMode,
		},
	}
}

func getServiceList() []Service {
	return []Service{
		{
			Name: serviceNameMain,
			Annotations: map[string]string{
				"role": "tc-main",
			},
			ServiceSpec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: servicePort, TargetPort: intstr.FromInt32(8111)},
				},
				Selector: map[string]string{
					"app.kubernetes.io/name": TeamCityName,
				},
			},
		},
		{
			Name: serviceNameSecondary,
			Annotations: map[string]string{
				"role": "tc-secondary",
			},
			ServiceSpec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: servicePort, TargetPort: intstr.FromInt32(8111)},
				},
				Selector: map[string]string{
					"app.kubernetes.io/name": TeamCityName,
				},
			},
		},
	}
}

func getLabels() map[string]string {
	return map[string]string{
		"foo":                    "bar",
		"teamcity":               "the best",
		"app.kubernetes.io/name": "label-override-test",
	}
}

func getIngressList() []Ingress {
	ingressClassName := "nginx"
	ingressServiceBackend := netv1.IngressServiceBackend{
		Name: serviceNameMain,
		Port: netv1.ServiceBackendPort{
			Number: servicePort,
		},
	}
	ingressHttpRuleValue := netv1.HTTPIngressRuleValue{
		Paths: []netv1.HTTPIngressPath{
			{
				Backend: netv1.IngressBackend{Service: &ingressServiceBackend},
			},
		},
	}
	ingressRuleValue := netv1.IngressRuleValue{
		HTTP: &ingressHttpRuleValue,
	}
	return []Ingress{
		{
			Name: ingressNameFirst,
			Annotations: map[string]string{
				"role": "tc-main",
			},
			IngressSpec: netv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []netv1.IngressRule{
					{
						Host:             TeamCityName + ".myteamcity.com",
						IngressRuleValue: ingressRuleValue,
					},
				},
			},
		},
		{
			Name: ingressNameSecondary,
			Annotations: map[string]string{
				"role": "tc-secondary",
			},
			IngressSpec: netv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []netv1.IngressRule{
					{
						Host:             TeamCityName + "-secondary.myteamcity.com",
						IngressRuleValue: ingressRuleValue,
					},
				},
			},
		},
	}
}

func getSecondaryNodes() []Node {
	return []Node{
		getNode("secondary-0", make(map[string]string), requests, []string{"foo"}),
		getNode("secondary-1", make(map[string]string), requests, []string{"bar"}),
	}
}

func getNode(nodeName string, annotations map[string]string, requests corev1.ResourceList, responsibilities []string) Node {
	return Node{
		Name:        nodeName,
		Annotations: annotations,
		Spec: NodeSpec{
			Requests:         requests,
			Responsibilities: responsibilities,
		},
	}
}

func getServiceAccount() ServiceAccount {
	return ServiceAccount{
		Name: "test-service-account",
		Annotations: map[string]string{
			"eks.amazonaws.com/role-arn": "some-role",
		},
	}
}
