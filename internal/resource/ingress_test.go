package resource

import (
	"context"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Ingress", func() {
	Context("TeamCity with ingress", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				DefaultClient = &ingressK8sClientMock{}
				teamcity.Spec.ServiceList = getServiceList()
				teamcity.Spec.IngressList = getIngressList()
			})
		})
		It("sets a list of objects with proper length, names, and namespaces", func() {
			objList, err := DefaultIngressBuilder.BuildObjectList()
			desiredIngressList := Instance.Spec.IngressList
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(len(desiredIngressList)))
			for idx, obj := range objList {
				ing := obj.(*netv1.Ingress)
				Expect(ing.Name).To(Equal(desiredIngressList[idx].Name))
				Expect(ing.Namespace).To(Equal(TeamCityNamespace))
			}
		})
		It("updates objects' configuration properly", func() {
			objList, err := DefaultIngressBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for idx, obj := range objList {
				err = DefaultIngressBuilder.Update(obj)
				Expect(err).NotTo(HaveOccurred())
				actual := obj.(*netv1.Ingress)
				expected := Instance.Spec.IngressList[idx]
				Expect(actual.Annotations).To(Equal(expected.Annotations))
				Expect(actual.Spec).To(Equal(expected.IngressSpec))
			}
		})
		It("returns obsolete objects correctly", func() {
			obsoleteObjects, err := DefaultIngressBuilder.GetObsoleteObjects(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(len(obsoleteObjects)).To(Equal(1))
			Expect(obsoleteObjects[0].GetName()).To(Equal(StaleIngressName))
		})
	})
})

type ingressK8sClientMock struct {
	client.Client
}

func (m *ingressK8sClientMock) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	listIngress, ok := list.(*netv1.IngressList)
	if !ok {
		return fmt.Errorf("unable to convert object list to ingress list")
	}
	existingIngressList := getIngressList()

	listIngress.Items = append(listIngress.Items, netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: existingIngressList[0].Name,
		},
	}, netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: existingIngressList[1].Name,
		},
	}, netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: StaleIngressName,
		},
	})
	return nil
}
