package predicate

import (
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ = Describe("Predicate", func() {
	Context("PersistentVolumeClaim", func() {
		var pvcPredicate predicate.Predicate
		BeforeEach(func() {
			pvcPredicate = PersistentVolumeClaimEventPredicates()
		})
		It("should filter update events correctly", func() {
			By("returning true when update event contains difference in Spec field", func() {
				storageClass1 := "standard"
				storageClass2 := "premium"
				result := pvcPredicate.Update(event.UpdateEvent{
					ObjectOld: &v12.PersistentVolumeClaim{Spec: v12.PersistentVolumeClaimSpec{StorageClassName: &storageClass1}},
					ObjectNew: &v12.PersistentVolumeClaim{Spec: v12.PersistentVolumeClaimSpec{StorageClassName: &storageClass2}},
				})
				Expect(result).To(Equal(true))
			})
			By("returning false when update event contains NO difference in Spec field", func() {
				storageClass := "standard"
				result := pvcPredicate.Update(event.UpdateEvent{
					ObjectOld: &v12.PersistentVolumeClaim{Spec: v12.PersistentVolumeClaimSpec{StorageClassName: &storageClass}},
					ObjectNew: &v12.PersistentVolumeClaim{Spec: v12.PersistentVolumeClaimSpec{StorageClassName: &storageClass}},
				})
				Expect(result).To(Equal(false))
			})
			By("returning false when old object is not a PVC", func() {
				storageClass := "standard"
				result := pvcPredicate.Update(event.UpdateEvent{
					ObjectOld: &v1.StatefulSet{},
					ObjectNew: &v12.PersistentVolumeClaim{Spec: v12.PersistentVolumeClaimSpec{StorageClassName: &storageClass}},
				})
				Expect(result).To(Equal(false))
			})
			By("returning false when new object is not a PVC", func() {
				storageClass := "standard"
				result := pvcPredicate.Update(event.UpdateEvent{
					ObjectOld: &v12.PersistentVolumeClaim{Spec: v12.PersistentVolumeClaimSpec{StorageClassName: &storageClass}},
					ObjectNew: &v1.StatefulSet{},
				})
				Expect(result).To(Equal(false))
			})
		})
		It("should filter create events correctly", func() {
			By("returning true when it's a create event", func() {
				result := pvcPredicate.Create(event.CreateEvent{
					Object: &v12.PersistentVolumeClaim{},
				})
				Expect(result).To(Equal(true))
			})
		})
		It("should filter delete events correctly", func() {
			By("returning false when delete state of an object is unknown", func() {
				result := pvcPredicate.Delete(event.DeleteEvent{
					DeleteStateUnknown: true,
				})
				Expect(result).To(Equal(false))
			})
			By("returning true when delete state of an object is known", func() {
				result := pvcPredicate.Delete(event.DeleteEvent{
					DeleteStateUnknown: false,
				})
				Expect(result).To(Equal(true))
			})
		})
		It("should filter generic events correctly", func() {
			By("returning true for generic events", func() {
				result := pvcPredicate.Generic(event.GenericEvent{
					Object: &v12.PersistentVolumeClaim{},
				})
				Expect(result).To(Equal(true))
			})
		})
	})
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
				Expect(result).To(Equal(true))
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
		It("should filter generic events correctly", func() {
			By("returning true for generic events", func() {
				result := predicate.Generic(event.GenericEvent{
					Object: &v1.StatefulSet{},
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
					Object: &TeamCity{},
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
			By("returning true when it's an update event", func() {
				result := predicate.Update(event.UpdateEvent{
					ObjectOld: &TeamCity{},
					ObjectNew: &TeamCity{},
				})
				Expect(result).To(Equal(true))
			})
		})
		It("should filter generic events correctly", func() {
			By("returning true for generic events", func() {
				result := predicate.Generic(event.GenericEvent{
					Object: &TeamCity{},
				})
				Expect(result).To(Equal(true))
			})
		})

	})

})
