package resource

import (
	"context"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ServiceAccount", func() {
	Context("TeamCity with serviceaccount", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				DefaultClient = &serviceAccountK8sClientMock{}
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
		It("returns obsolete objects correctly", func() {
			obsoleteObjects, err := DefaultServiceAccountBuilder.GetObsoleteObjects(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(len(obsoleteObjects)).To(Equal(1))
			Expect(obsoleteObjects[0].GetName()).To(Equal(StaleServiceAccountName))
		})
	})
})

type serviceAccountK8sClientMock struct {
	client.Client
}

func (m *serviceAccountK8sClientMock) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	listServiceAccount, ok := list.(*v1.ServiceAccountList)
	if !ok {
		return fmt.Errorf("unable to convert object list to ingress list")
	}
	sa := getServiceAccount()

	listServiceAccount.Items = []v1.ServiceAccount{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: sa.Name,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: StaleServiceAccountName,
			},
		},
	}
	return nil
}
