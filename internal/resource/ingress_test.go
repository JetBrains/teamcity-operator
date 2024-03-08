package resource

//import (
//	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
//	. "github.com/onsi/ginkgo/v2"
//	. "github.com/onsi/gomega"
//	netv1 "k8s.io/api/networking/v1"
//)
//
//var _ = Describe("Ingress", func() {
//	Context("TeamCity with ingress", func() {
//		BeforeEach(func() {
//			BeforeEachBuild(func(teamcity *TeamCity) {
//				teamcity.Spec.ServiceList = getServiceList()
//				teamcity.Spec.IngressList = getIngressList()
//			})
//		})
//		It("sets a list of objects with proper length, names, and namespaces", func() {
//			objList, err := DefaultIngressBuilder.BuildObjectList()
//			desiredIngressList := Instance.Spec.IngressList
//			Expect(err).NotTo(HaveOccurred())
//			Expect(len(objList)).To(Equal(len(desiredIngressList)))
//			for idx, obj := range objList {
//				ing := obj.(*netv1.Ingress)
//				Expect(ing.Name).To(Equal(desiredIngressList[idx].Name))
//				Expect(ing.Namespace).To(Equal(TeamCityNamespace))
//			}
//		})
//		It("updates objects' configuration properly", func() {
//			objList, err := DefaultIngressBuilder.BuildObjectList()
//			Expect(err).NotTo(HaveOccurred())
//			for idx, obj := range objList {
//				err = DefaultIngressBuilder.Update(obj)
//				Expect(err).NotTo(HaveOccurred())
//				actual := obj.(*netv1.Ingress)
//				expected := Instance.Spec.IngressList[idx]
//				Expect(actual.Annotations).To(Equal(expected.Annotations))
//				Expect(actual.Spec).To(Equal(expected.IngressSpec))
//			}
//		})
//	})
//})
