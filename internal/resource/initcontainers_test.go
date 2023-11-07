package resource

import (
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("StatefulSetWithInitContainers", func() {
	Describe("Build", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *v1alpha1.TeamCity) {
				teamcity.Spec.InitContainers = getInitContainers()
			})
		})
		It("adds init containers", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			sts := obj.(*v1.StatefulSet)
			Expect(sts.Spec.Template.Spec.InitContainers).To(Equal(getInitContainers()))
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			BeforeEachUpdate(func(teamcity *v1alpha1.TeamCity) {
				teamcity.Spec.InitContainers = getInitContainers()
			})
		})
		It("adds init containers after update", func() {
			statefulSet := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Instance.Name,
					Namespace: Instance.Namespace,
				},
			}
			Expect(DefaultStatefulSetBuilder.Update(statefulSet)).To(Succeed())
			Expect(len(statefulSet.OwnerReferences)).To(Equal(1))
			Expect(statefulSet.Spec.Template.Spec.InitContainers).To(Equal(getInitContainers()))
		})
	})
})

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
