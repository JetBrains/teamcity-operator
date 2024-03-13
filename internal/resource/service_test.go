package resource

import (
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
)

var _ = Describe("Service", func() {
	Context("TeamCity with service", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.ServiceList = getServiceList()
			})
		})
		It("sets a list of objects with proper length, names, and namespaces", func() {
			objList, err := DefaultServiceBuilder.BuildObjectList()
			desiredServiceList := Instance.Spec.ServiceList
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(len(desiredServiceList)))
			for idx, obj := range objList {
				svc := obj.(*v12.Service)
				Expect(svc.Name).To(Equal(desiredServiceList[idx].Name))
				Expect(svc.Namespace).To(Equal(TeamCityNamespace))
			}
		})
		It("updates objects' configuration properly", func() {
			objList, err := DefaultServiceBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for idx, obj := range objList {
				err = DefaultServiceBuilder.Update(obj)
				Expect(err).NotTo(HaveOccurred())
				actual := obj.(*v12.Service)
				expected := Instance.Spec.ServiceList[idx]
				Expect(actual.Annotations).To(Equal(expected.Annotations))
				Expect(actual.Spec).To(Equal(expected.ServiceSpec))
			}
		})
	})
})
