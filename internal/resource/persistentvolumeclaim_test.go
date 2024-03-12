package resource

//
//import (
//	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
//	. "github.com/onsi/ginkgo/v2"
//	. "github.com/onsi/gomega"
//	v12 "k8s.io/api/core/v1"
//)
//
//var _ = Describe("PersistentVolumeClaim", func() {
//	Context("TeamCity with required persistence", func() {
//		BeforeEach(func() {
//			BeforeEachBuild(func(teamcity *TeamCity) {})
//		})
//		It("creates a single PVC with correct settings", func() {
//			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
//			Expect(err).NotTo(HaveOccurred())
//			Expect(len(objList)).To(Equal(1))
//			obj := objList[0]
//			err = DefaultPersistentVolumeClaimBuilder.Update(obj)
//			Expect(err).NotTo(HaveOccurred())
//			actual := obj.(*v12.PersistentVolumeClaim)
//			expected := Instance.Spec.DataDirVolumeClaim
//			Expect(actual.Spec).To(Equal(expected.Spec))
//			Expect(actual.Annotations).To(Equal(expected.Annotations))
//		})
//	})
//
//	Context("TeamCity with additional persistence", func() {
//		BeforeEach(func() {
//			BeforeEachBuild(func(teamcity *TeamCity) {
//				teamcity.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{getAdditionalPVC()}
//			})
//		})
//		It("creates data dir PVC and one additional PVC with correct settings", func() {
//			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
//			Expect(err).NotTo(HaveOccurred())
//			Expect(len(objList)).To(Equal(2))
//			dataDirPersistentVolumeClaim := objList[0]
//			err = DefaultPersistentVolumeClaimBuilder.Update(dataDirPersistentVolumeClaim)
//			Expect(err).NotTo(HaveOccurred())
//			actual := dataDirPersistentVolumeClaim.(*v12.PersistentVolumeClaim)
//			expected := Instance.Spec.DataDirVolumeClaim
//			Expect(actual.Spec).To(Equal(expected.Spec))
//
//			additionalPersistentVolumeClaim := objList[1]
//			err = DefaultPersistentVolumeClaimBuilder.Update(additionalPersistentVolumeClaim)
//			Expect(err).NotTo(HaveOccurred())
//			actual = additionalPersistentVolumeClaim.(*v12.PersistentVolumeClaim)
//			expected = Instance.Spec.PersistentVolumeClaims[0]
//			Expect(actual.Spec).To(Equal(expected.Spec))
//			Expect(actual.Annotations).To(Equal(expected.Annotations))
//		})
//	})
//})
