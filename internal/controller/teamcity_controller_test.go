package controller

import (
	"context"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("TeamCity controller", func() {

	Context("TeamCity with minimum configuration", func() {
		const (
			TeamCityName      = "test"
			TeamCityNamespace = "default"
			TeamCityImage     = "jetbrains/fetchTeamcity-server:latest"
		)

		var TeamCityReplicas = int32(0)
		var teamcity *TeamCity
		var (
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
			pvc = CustomPersistentVolumeClaim{
				Name: pvcName,
				Spec: pvcSpec,
			}
			requests = corev1.ResourceList{
				"cpu":    resource.MustParse("1"),
				"memory": resource.MustParse("1000"),
			}
		)

		BeforeEach(func() {
			teamcity = &TeamCity{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "jetbrains.com/v1beta1",
					Kind:       "TeamCity",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: TeamCitySpec{
					Image:              TeamCityImage,
					Replicas:           &TeamCityReplicas,
					Requests:           requests,
					DataDirVolumeClaim: pvc,
					InitContainers:     getInitContainers(),
				},
			}
			Expect(k8sClient.Create(ctx, teamcity)).To(Succeed())
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, teamcity)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: teamcity.Name, Namespace: teamcity.Namespace}, teamcity)
				return errors.IsNotFound(err)
			}, 5).Should(BeTrue())
		})

		It("should successfully reconcile an operand", func() {
			By("setting operand properties correctly", func() {
				fetchedTeamCity := &TeamCity{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: TeamCityName, Namespace: TeamCityNamespace}, fetchedTeamCity)).To(Succeed())
				Expect(fetchedTeamCity.Spec.Image).To(Equal(TeamCityImage))
				Expect(fetchedTeamCity.Spec.Replicas).To(Equal(&TeamCityReplicas))
			})
			By("creating statefulset with operand properties", func() {
				var producedStatefulSet *v1.StatefulSet
				producedStatefulSet = statefulSet(ctx, teamcity)
				Expect(producedStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(TeamCityImage))
				Expect(producedStatefulSet.Spec.Replicas).To(Equal(&TeamCityReplicas))
			})
			By("adds init containers", func() {
				var producedStatefulSet *v1.StatefulSet
				producedStatefulSet = statefulSet(ctx, teamcity)
				Expect(producedStatefulSet.Spec.Template.Spec.InitContainers).To(Equal(getInitContainers()))
			})
		})
	})

	Context("TeamCity with database configurations", func() {
		const (
			TeamCityName               = "test-with-database"
			TeamCityNamespace          = "default"
			TeamCityImage              = "jetbrains/fetchTeamcity-server:latest"
			TeamCityDatabaseSecretName = "database-secret"
		)
		var (
			teamcity       *TeamCity
			databaseSecret = DatabaseSecret{
				Secret: TeamCityDatabaseSecretName,
			}
			databaseProperties *corev1.Secret

			pvcAccessMode       = []corev1.PersistentVolumeAccessMode{"ReadWriteMany"}
			pvcStorageClassName = "standard"
			pvcVolumeMode       = corev1.PersistentVolumeFilesystem
			pvcResources        = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			}
			dataDirPVCName = "test-data-dir-pvc"
			pvcSpec        = corev1.PersistentVolumeClaimSpec{
				AccessModes:      pvcAccessMode,
				StorageClassName: &pvcStorageClassName,
				VolumeMode:       &pvcVolumeMode,
				Resources:        pvcResources,
			}
			dataDirPVC = CustomPersistentVolumeClaim{
				Name:        dataDirPVCName,
				Spec:        pvcSpec,
				VolumeMount: corev1.VolumeMount{MountPath: "/storage"},
			}

			configDirPVCName = "test-config-dir-pvc"
			configDirPVC     = CustomPersistentVolumeClaim{
				Name:        configDirPVCName,
				VolumeMount: corev1.VolumeMount{MountPath: "/storage/config"},
				Spec:        pvcSpec,
			}

			TeamCityReplicas = int32(0)
			requests         = corev1.ResourceList{
				"cpu":    resource.MustParse("1"),
				"memory": resource.MustParse("1000"),
			}
		)
		BeforeEach(func() {
			databaseProperties = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityDatabaseSecretName,
					Namespace: TeamCityNamespace,
				},
				Data: map[string][]byte{
					"database.properties": []byte("connectionUrl=jdbc:mysql://mysql.default:3306/fetchTeamcity\nconnectionProperties.user=root\nconnectionProperties.password=password"),
				},
			}
			teamcity = &TeamCity{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "jetbrains.com/v1beta1",
					Kind:       "TeamCity",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: TeamCitySpec{
					Image:                  TeamCityImage,
					DataDirVolumeClaim:     dataDirPVC,
					PersistentVolumeClaims: []CustomPersistentVolumeClaim{configDirPVC},
					Replicas:               &TeamCityReplicas,
					Requests:               requests,
					DatabaseSecret:         databaseSecret,
					InitContainers:         getInitContainers(),
				},
			}
			Expect(k8sClient.Create(ctx, databaseProperties)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: TeamCityDatabaseSecretName, Namespace: TeamCityNamespace}, databaseProperties)
				return errors.IsNotFound(err)
			}, 20).Should(BeFalse())
			Expect(k8sClient.Create(ctx, teamcity)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: TeamCityName, Namespace: TeamCityNamespace}, teamcity)
				return errors.IsNotFound(err)
			}, 20).Should(BeFalse())
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, databaseProperties)).To(Succeed())
			Expect(k8sClient.Delete(ctx, teamcity)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: teamcity.Name, Namespace: teamcity.Namespace}, teamcity)
				return errors.IsNotFound(err)
			}, 5).Should(BeTrue())
		})

		It("should successfully reconcile an operand with given secret", func() {
			By("creating statefulset with mounted secret", func() {
				var producedStatefulSet *v1.StatefulSet
				producedStatefulSet = statefulSet(ctx, teamcity)
				Expect(producedStatefulSet).ToNot(BeNil())
				teamcityContainer := producedStatefulSet.Spec.Template.Spec.Containers[0]
				Expect(len(teamcityContainer.VolumeMounts)).To(Equal(3))
			})
			By("adds init containers", func() {
				var producedStatefulSet *v1.StatefulSet
				producedStatefulSet = statefulSet(ctx, teamcity)
				Expect(producedStatefulSet.Spec.Template.Spec.InitContainers).To(Equal(getInitContainers()))
			})
		})
	})
})

func statefulSet(ctx context.Context, teamcity *TeamCity) (statefulSet *v1.StatefulSet) {
	EventuallyWithOffset(1, func() error {
		var err error
		statefulSet, err = clientSet.AppsV1().StatefulSets(teamcity.Namespace).Get(ctx, teamcity.Name, metav1.GetOptions{})
		return err
	}, 10).Should(Succeed())
	return statefulSet
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
			TerminationMessagePath:   "/dev/termination-log",
			TerminationMessagePolicy: "File",
			ImagePullPolicy:          "Always",
		},
	}
}
