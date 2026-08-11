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

var _ = Describe("PersistentVolumeClaim", func() {
	Context("TeamCity with required persistence", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				DefaultClient = &pvcK8sClientMock{}
			})
		})
		It("creates a single PVC with correct settings", func() {
			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(1))
			obj := objList[0]
			err = DefaultPersistentVolumeClaimBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			actual := obj.(*v12.PersistentVolumeClaim)
			expected := Instance.Spec.DataDirVolumeClaim
			Expect(actual.Spec).To(Equal(expected.Spec))
			Expect(actual.Annotations).To(Equal(expected.Annotations))
		})
	})

	Context("TeamCity with additional persistence", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{getAdditionalPVC()}
			})
		})
		It("creates data dir PVC and one additional PVC with correct settings", func() {
			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(2))
			dataDirPersistentVolumeClaim := objList[0]
			err = DefaultPersistentVolumeClaimBuilder.Update(dataDirPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			actual := dataDirPersistentVolumeClaim.(*v12.PersistentVolumeClaim)
			expected := Instance.Spec.DataDirVolumeClaim
			Expect(actual.Spec).To(Equal(expected.Spec))

			additionalPersistentVolumeClaim := objList[1]
			err = DefaultPersistentVolumeClaimBuilder.Update(additionalPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			actual = additionalPersistentVolumeClaim.(*v12.PersistentVolumeClaim)
			expected = Instance.Spec.PersistentVolumeClaims[0]
			Expect(actual.Spec).To(Equal(expected.Spec))
			Expect(actual.Annotations).To(Equal(expected.Annotations))
		})
		It("returns obsolete objects correctly", func() {
			obsoleteObjects, err := DefaultPersistentVolumeClaimBuilder.GetObsoleteObjects(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(len(obsoleteObjects)).To(Equal(1))
			Expect(obsoleteObjects[0].GetName()).To(Equal(StalePvcName))
		})
	})

	Context("TeamCity with node data dir greenfield", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = []Node{getSecondaryNode()}
				teamcity.Spec.NodeDataDirVolumeClaim = getNodeDataDirTemplate()
			})
		})
		It("creates one node-data PVC per node without claim overrides", func() {
			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(3))

			names := []string{objList[0].GetName(), objList[1].GetName(), objList[2].GetName()}
			Expect(names).To(ContainElement(Instance.Spec.DataDirVolumeClaim.Name))
			Expect(names).To(ContainElement("node-data-dir-" + Instance.Spec.MainNode.Name))
			Expect(names).To(ContainElement("node-data-dir-" + Instance.Spec.SecondaryNodes[0].Name))

			for _, obj := range objList[1:] {
				err = DefaultPersistentVolumeClaimBuilder.Update(obj)
				Expect(err).NotTo(HaveOccurred())
				actual := obj.(*v12.PersistentVolumeClaim)
				Expect(actual.Spec.Resources).To(Equal(Instance.Spec.NodeDataDirVolumeClaim.Spec.Resources))
			}
		})
	})

	Context("TeamCity with node data dir adopt", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.NodeDataDirVolumeClaim = getNodeDataDirTemplate()
				teamcity.Spec.MainNode.Spec.NodeDataDirClaimName = "existing-main-node-data"
			})
		})
		It("does not create a PVC for nodes with claim overrides", func() {
			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(objList)).To(Equal(1))
			Expect(objList[0].GetName()).To(Equal(Instance.Spec.DataDirVolumeClaim.Name))
		})
	})

	Context("node data dir PVC retention", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.NodeDataDirVolumeClaim = getNodeDataDirTemplate()
				DefaultClient = &pvcK8sClientMockWithNodeData{}
			})
		})
		It("does not mark labeled node-data PVCs obsolete after feature disable", func() {
			Instance.Spec.NodeDataDirVolumeClaim = nil
			builder = &TeamCityResourceBuilder{
				Instance: &Instance,
				Scheme:   scheme,
				Client:   DefaultClient,
			}
			DefaultPersistentVolumeClaimBuilder = builder.PersistentVolumeClaim()

			obsoleteObjects, err := DefaultPersistentVolumeClaimBuilder.GetObsoleteObjects(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(obsoleteObjects)).To(Equal(1))
			Expect(obsoleteObjects[0].GetName()).To(Equal(StalePvcName))
		})
		It("labels managed node-data PVCs", func() {
			objList, err := DefaultPersistentVolumeClaimBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			var nodeDataObj client.Object
			for _, obj := range objList {
				if obj.GetName() == "node-data-dir-"+Instance.Spec.MainNode.Name {
					nodeDataObj = obj
					break
				}
			}
			Expect(nodeDataObj).NotTo(BeNil())
			Expect(DefaultPersistentVolumeClaimBuilder.Update(nodeDataObj)).To(Succeed())
			Expect(nodeDataObj.GetLabels()["teamcity.jetbrains.com/node-data-dir"]).To(Equal("true"))
		})
	})
})

type pvcK8sClientMock struct {
	client.Client
}

func (m *pvcK8sClientMock) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	listPvc, ok := list.(*v12.PersistentVolumeClaimList)
	if !ok {
		return fmt.Errorf("unable to convert object list to pvc list")
	}
	existingPvc := getAdditionalPVC()

	listPvc.Items = append(listPvc.Items, v12.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: existingPvc.Name,
		},
	}, v12.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: StalePvcName,
		},
	})
	return nil
}

type pvcK8sClientMockWithNodeData struct {
	client.Client
}

func (m *pvcK8sClientMockWithNodeData) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	listPvc, ok := list.(*v12.PersistentVolumeClaimList)
	if !ok {
		return fmt.Errorf("unable to convert object list to pvc list")
	}
	listPvc.Items = append(listPvc.Items, v12.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-data-dir-main-node",
			Labels: map[string]string{
				"app.kubernetes.io/name":               TeamCityName,
				"app.kubernetes.io/component":          "teamcity-server",
				"app.kubernetes.io/part-of":            "teamcity",
				"teamcity.jetbrains.com/node-data-dir": "true",
			},
		},
	}, v12.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: StalePvcName,
			Labels: map[string]string{
				"app.kubernetes.io/name":      TeamCityName,
				"app.kubernetes.io/component": "teamcity-server",
				"app.kubernetes.io/part-of":   "teamcity",
			},
		},
	})
	return nil
}
