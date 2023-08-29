package controller

import (
	"context"
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("TeamCity controller", func() {
	const (
		TeamCityName      = "test"
		TeamCityNamespace = "default"
		TeamCityImage     = "jetbrains/teamcity-server:latest"
	)
	var TeamCityReplicas = int32(0)
	var teamcity *v1alpha1.TeamCity
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
		pvc = v1alpha1.CustomPersistentVolumeClaim{
			Name: pvcName,
			Spec: pvcSpec,
		}
		requests = corev1.ResourceList{
			"cpu":    resource.MustParse("1"),
			"memory": resource.MustParse("1000"),
		}
	)

	Context("TeamCity controller test", func() {
		BeforeEach(func() {
			teamcity = &v1alpha1.TeamCity{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "jetbrains.com/v1alpha1",
					Kind:       "TeamCity",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image:                  TeamCityImage,
					Replicas:               &TeamCityReplicas,
					Requests:               requests,
					PersistentVolumeClaims: []v1alpha1.CustomPersistentVolumeClaim{pvc},
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
				fetchedTeamCity := &v1alpha1.TeamCity{}
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
		})
	})
})

func statefulSet(ctx context.Context, teamcity *v1alpha1.TeamCity) *v1.StatefulSet {
	var statefulSet *v1.StatefulSet
	EventuallyWithOffset(1, func() error {
		var err error
		statefulSet, err = clientSet.AppsV1().StatefulSets(teamcity.Namespace).Get(ctx, teamcity.Name, metav1.GetOptions{})
		return err
	}, 10).Should(Succeed())
	return statefulSet
}
