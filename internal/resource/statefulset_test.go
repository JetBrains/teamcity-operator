package resource

import (
	v1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
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
	)
	Describe("Build", func() {
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
