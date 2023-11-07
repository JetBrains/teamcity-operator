package predicate

import (
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ = Describe("Predicate", func() {
	Context("StatefulSet", func() {
		var predicate predicate.Predicate
		BeforeEach(func() {
			predicate = StatefulSetEventPredicates()
		})
		It("should filter update events correctly", func() {
			By("returning true when update event contains difference in Spec field", func() {
				result := predicate.Update(event.UpdateEvent{
					ObjectOld: &v1.StatefulSet{Spec: v1.StatefulSetSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "nginx"}}}}}},
					ObjectNew: &v1.StatefulSet{Spec: v1.StatefulSetSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "ngins"}}}}}},
				})
				Expect(result).To(Equal(false))
			})
			By("returning false when update event contains NO difference in Spec field", func() {
				result := predicate.Update(event.UpdateEvent{
					ObjectOld: &v1.StatefulSet{Spec: v1.StatefulSetSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "nginx"}}}}}},
					ObjectNew: &v1.StatefulSet{Spec: v1.StatefulSetSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "nginx"}}}}}},
				})
				Expect(result).To(Equal(false))
			})
			By("returning false when update event has different objects", func() {
				result := predicate.Update(event.UpdateEvent{
					ObjectOld: &v1.StatefulSet{Spec: v1.StatefulSetSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "nginx"}}}}}},
					ObjectNew: &v1.Deployment{Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "nginx"}}}}}},
				})
				Expect(result).To(Equal(false))
			})
		})
		It("should filter create events correctly", func() {
			By("returning true when it's a create event", func() {
				result := predicate.Create(event.CreateEvent{
					Object: &v1.StatefulSet{Spec: v1.StatefulSetSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{Containers: []v12.Container{{Image: "nginx"}}}}}},
				})
				Expect(result).To(Equal(true))
			})
		})
		It("should filter delete events correctly", func() {
			By("returning false when delete state of an object is unknown", func() {
				result := predicate.Delete(event.DeleteEvent{
					DeleteStateUnknown: true,
				})
				Expect(result).To(Equal(false))
			})
			By("returning false when delete state of an object is known", func() {
				result := predicate.Delete(event.DeleteEvent{
					DeleteStateUnknown: false,
				})
				Expect(result).To(Equal(true))
			})
		})
	})
	Context("TeamCity", func() {
		var predicate predicate.Predicate
		BeforeEach(func() {
			predicate = TeamcityEventPredicates()
		})
		It("should filter create events  correctly", func() {
			By("returning true when it's a create event", func() {
				result := predicate.Create(event.CreateEvent{
					Object: &v1alpha1.TeamCity{},
				})
				Expect(result).To(Equal(true))
			})
		})
		It("should filter delete events correctly", func() {
			By("returning false when delete state of an object is unknown", func() {
				result := predicate.Delete(event.DeleteEvent{
					DeleteStateUnknown: true,
				})
				Expect(result).To(Equal(false))
			})
			By("returning false when delete state of an object is known", func() {
				result := predicate.Delete(event.DeleteEvent{
					DeleteStateUnknown: false,
				})
				Expect(result).To(Equal(true))
			})
		})
		It("should filter update events correctly", func() {
			By("returning true when it's a create event", func() {
				result := predicate.Update(event.UpdateEvent{
					ObjectOld: &v1alpha1.TeamCity{},
					ObjectNew: &v1alpha1.TeamCity{},
				})
				Expect(result).To(Equal(true))
			})
		})

	})

})
