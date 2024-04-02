package resource

import (
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("ServiceAccount", func() {
	Context("TeamCity with serviceaccount", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.ServiceAccount = getServiceAccount()
			})
		})
		It("sets a list of objects with proper length, names, and namespaces", func() {
			objList, err := DefaultServiceAccountBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(1))
			obj := objList[0]
			sa := obj.(*v1.ServiceAccount)
			Expect(sa.Name).To(Equal(getServiceAccount().Name))
			Expect(sa.Namespace).To(Equal(TeamCityNamespace))
		})
		It("updates objects' configuration properly", func() {
			objList, err := DefaultServiceAccountBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			obj := objList[0]
			err = DefaultServiceAccountBuilder.Update(obj)
			sa := obj.(*v1.ServiceAccount)
			expected := getServiceAccount()
			Expect(sa.Annotations).To(Equal(expected.Annotations))
		})
	})
})
