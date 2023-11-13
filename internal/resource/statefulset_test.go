package resource

import (
	"fmt"
	v1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"strings"
)

var _ = Describe("StatefulSet", func() {
	var (
		instance           v1alpha1.TeamCity
		statefulSetBuilder *StatefulSetBuilder
		scheme             *runtime.Scheme
		builder            *TeamCityResourceBuilder
		TeamCityReplicas   = int32(0)

		pvcAccessMode       = []corev1.PersistentVolumeAccessMode{"ReadWriteMany"}
		pvcStorageClassName = "standard"
		pvcVolumeMode       = corev1.PersistentVolumeFilesystem
		pvcResources        = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		}
		pvcName = "test-dataDirPvc"
		pvcSpec = corev1.PersistentVolumeClaimSpec{
			AccessModes:      pvcAccessMode,
			StorageClassName: &pvcStorageClassName,
			VolumeMode:       &pvcVolumeMode,
			Resources:        pvcResources,
		}
		dataDirPvc = v1alpha1.CustomPersistentVolumeClaim{
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
		startupConfig = map[string]string{
			"foo":   "bar",
			"hello": "world",
		}
	)
	Context("TeamCity with minimal configuration", func() {
		BeforeEach(func() {
			statefulSetBuilder = builder.StatefulSet()
			instance = v1alpha1.TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image:                  TeamCityImage,
					Replicas:               &TeamCityReplicas,
					PersistentVolumeClaims: []v1alpha1.CustomPersistentVolumeClaim{dataDirPvc},
					Requests:               requests,
					XmxPercentage:          xmxPercentage,
				},
			}
			scheme = runtime.NewScheme()
			Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
			Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())
			builder = &TeamCityResourceBuilder{
				Instance: &instance,
				Scheme:   scheme,
			}
			statefulSetBuilder = builder.StatefulSet()

		})
		It("sets a name and namespace", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			sts := obj.(*v1.StatefulSet)
			Expect(sts.Name).To(Equal(TeamCityName))
			Expect(sts.Namespace).To(Equal(TeamCityNamespace))
		})
		It("adds the correct label selector", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			labels := statefulSet.Spec.Selector.MatchLabels
			Expect(labels["app.kubernetes.io/name"]).To(Equal(instance.Name))
		})
		It("sets required resources requests for container", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = statefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)
			expected := corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    builder.Instance.Spec.Requests["cpu"],
					"memory": builder.Instance.Spec.Requests["memory"],
				},
				Limits: nil,
			}
			actual := statefulSet.Spec.Template.Spec.Containers[0].Resources
			Expect(actual).To(Equal(expected))
		})
		It("sets prestop command for container", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = statefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)
			expected := []string{"/bin/sh", "-c", "/opt/teamcity/bin/shutdown.sh"}
			actual := statefulSet.Spec.Template.Spec.Containers[0].Lifecycle.PreStop.Exec.Command
			Expect(actual).To(Equal(expected))
		})
		It("calculates and provides env vars correctly", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = statefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)
			xmxValue := xmxValueCalculator(xmxPercentage, builder.Instance.Spec.Requests.Memory().Value())
			datadirPath := volumeMountsBuilder(builder.Instance)[0].MountPath

			dataPath := corev1.EnvVar{
				Name:  "TEAMCITY_DATA_PATH",
				Value: datadirPath,
			}

			logsPath := corev1.EnvVar{
				Name:  "TEAMCITY_LOGS_PATH",
				Value: fmt.Sprintf("%s/%s", datadirPath, "logs"),
			}

			memOpts := corev1.EnvVar{
				Name:  "TEAMCITY_SERVER_MEM_OPTS",
				Value: fmt.Sprintf("%s%s", "-Xmx", xmxValue),
			}

			serverOpts := corev1.EnvVar{
				Name: "TEAMCITY_SERVER_OPTS",
				Value: "-XX:+HeapDumpOnOutOfMemoryError -XX:+DisableExplicitGC" +
					fmt.Sprintf(" -XX:HeapDumpPath=%s%s%s", datadirPath, "/memoryDumps/", TeamCityName) +
					fmt.Sprintf(" -Dteamcity.server.nodeId=%s", TeamCityName) + fmt.Sprintf(" -Dteamcity.server.rootURL=%s", TeamCityName)}
			expected := append([]corev1.EnvVar{}, memOpts, dataPath, logsPath, serverOpts)
			actual := statefulSet.Spec.Template.Spec.Containers[0].Env
			envVarsAreEqual := assert.ElementsMatch(GinkgoT(), expected, actual)
			Expect(envVarsAreEqual).To(Equal(true))
		})
		It("sets the owner reference", func() {
			statefulSet := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Name,
					Namespace: instance.Namespace,
				},
			}
			Expect(statefulSetBuilder.Update(statefulSet)).To(Succeed())
			Expect(len(statefulSet.OwnerReferences)).To(Equal(1))
			Expect(statefulSet.OwnerReferences[0].Name).To(Equal(builder.Instance.Name))
		})
		It("creates the required PersistentVolumeClaims", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			expected := []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pvcName,
						Namespace: builder.Instance.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "jetbrains.com/v1alpha1",
								Kind:               "TeamCity",
								Name:               builder.Instance.GetName(),
								UID:                builder.Instance.GetUID(),
								BlockOwnerDeletion: pointer.Bool(true),
								Controller:         pointer.Bool(true),
							},
						},
					},
					Spec: pvcSpec,
				},
			}
			actual := statefulSet.Spec.VolumeClaimTemplates
			Expect(actual).To(Equal(expected))
		})
	})
	Context("TeamCity with database properties", func() {
		BeforeEach(func() {
			statefulSetBuilder = builder.StatefulSet()
			instance = v1alpha1.TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image:                  TeamCityImage,
					Replicas:               &TeamCityReplicas,
					PersistentVolumeClaims: []v1alpha1.CustomPersistentVolumeClaim{dataDirPvc},
					Requests:               requests,
					DatabaseSecret:         databaseSecret,
				},
			}
			scheme = runtime.NewScheme()
			Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
			Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())
			builder = &TeamCityResourceBuilder{
				Instance: &instance,
				Scheme:   scheme,
			}
			statefulSetBuilder = builder.StatefulSet()

		})
		It("mounts database secret correctly", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = statefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			volumes := statefulSet.Spec.Template.Spec.Volumes
			Expect(len(volumes)).To(Equal(1))

			databaseSecretVolume := volumes[0]
			Expect(databaseSecretVolume.Name).To(Equal(DATABASE_PROPERTIES_VOLUME_NAME))
			Expect(databaseSecretVolume.Secret.SecretName).To(Equal(instance.Spec.DatabaseSecret.Secret))

			volumeMounts := statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts
			Expect(len(volumeMounts)).To(Equal(2))

			databaseSecretVolumeMount := volumeMounts[1]
			datadirPath := volumeMountsBuilder(builder.Instance)[0].MountPath
			datadirPathClean := strings.Replace(datadirPath, "/", "", -1)

			databaseSecretPathSplit := RemoveEmptyStrings(strings.Split(databaseSecretVolumeMount.MountPath, "/"))
			Expect(databaseSecretPathSplit).To(Equal([]string{datadirPathClean, "config", TEAMCITY_DATABASE_PROPERTIES_SUB_PATH}))
		})
	})
	Context("TeamCity with startup properties", func() {
		BeforeEach(func() {
			statefulSetBuilder = builder.StatefulSet()
			instance = v1alpha1.TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image:                   TeamCityImage,
					Replicas:                &TeamCityReplicas,
					PersistentVolumeClaims:  []v1alpha1.CustomPersistentVolumeClaim{dataDirPvc},
					Requests:                requests,
					StartupPropertiesConfig: startupConfig,
				},
			}
			scheme = runtime.NewScheme()
			Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
			Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())
			builder = &TeamCityResourceBuilder{
				Instance: &instance,
				Scheme:   scheme,
			}
			statefulSetBuilder = builder.StatefulSet()

		})
		It("sets serveropts correctly", func() {
			obj, err := statefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = statefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			containerEnv := statefulSet.Spec.Template.Spec.Containers[0].Env
			serverOptsEnvVarIndex := slices.IndexFunc(containerEnv, func(c corev1.EnvVar) bool { return c.Name == "TEAMCITY_SERVER_OPTS" })
			serverOpts := containerEnv[serverOptsEnvVarIndex].Value
			serverOptsSplit := strings.Fields(serverOpts)
			startUpServerOpts := serverOptsSplit[len(serverOptsSplit)-len(startupConfig):] //get elements of the split that correspond to startup vars

			i := 0
			for k, v := range startupConfig {
				expectedValue := fmt.Sprintf("-D%s=%s", k, v)
				Expect(expectedValue).To(Equal(startUpServerOpts[i]))
				i += 1
			}
		})
	})

})

func RemoveEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
