package resource

import (
	"context"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Service", func() {
	Context("TeamCity with service", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				DefaultClient = &serviceK8sClientMock{}
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
		It("returns obsolete objects correctly", func() {
			obsoleteObjects, err := DefaultServiceBuilder.GetObsoleteObjects(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(len(obsoleteObjects)).To(Equal(1))
			Expect(obsoleteObjects[0].GetName()).To(Equal(staleServiceName))
		})
	})
})

type serviceK8sClientMock struct {
	client.Client
}

func (m *serviceK8sClientMock) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	listService, ok := list.(*v12.ServiceList)
	if !ok {
		return fmt.Errorf("unable to convert object list to service list")
	}
	existingServiceList := getServiceList()

	listService.Items = append(listService.Items, v12.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: existingServiceList[0].Name,
		},
		Spec: v12.ServiceSpec{},
	}, v12.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: existingServiceList[1].Name,
		},
		Spec: v12.ServiceSpec{},
	}, v12.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: staleServiceName,
		},
		Spec: v12.ServiceSpec{},
	})
	return nil
}
