package resource

import (
	v1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
)

var _ = Describe("StatefulSet", func() {
	const (
		TeamCityName      = "test"
		TeamCityNamespace = "default"
		TeamCityImage     = "jetbrains/teamcity-server:latest"
	)
	var (
		instance           v1alpha1.TeamCity
		scheme             *runtime.Scheme
		builder            *TeamCityResourceBuilder
		statefulSetBuilder *StatefulSetBuilder
		TeamCityReplicas   = int32(0)

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
	)
	Describe("Build", func() {
		BeforeEach(func() {
			instance = v1alpha1.TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image:                  TeamCityImage,
					Replicas:               &TeamCityReplicas,
					PersistentVolumeClaims: []v1alpha1.CustomPersistentVolumeClaim{pvc},
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
			println("expected: ", expected)
			actual := statefulSet.Spec.VolumeClaimTemplates
			Expect(actual).To(Equal(expected))
		})
	})

	Describe("Update", func() {
		var statefulSetBuilder *StatefulSetBuilder
		BeforeEach(func() {
			instance = v1alpha1.TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image:    TeamCityImage,
					Replicas: &TeamCityReplicas,
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
	})
})
