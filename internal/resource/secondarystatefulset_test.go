package resource

import (
	"context"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var _ = Describe("Secondary StatefulSet", func() {
	Context("TeamCity secondary node with minimal configuration", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				DefaultClient = &secondaryStatefulSetK8sClientMock{}
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
			})
		})
		It("sets a name and namespace", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			desiredSecondaryNodes := getSecondaryNodes()
			for id, object := range objectList {
				statefulSet := object.(*v1.StatefulSet)
				Expect(statefulSet.Name).To(Equal(desiredSecondaryNodes[id].Name))
				Expect(statefulSet.Namespace).To(Equal(TeamCityNamespace))
			}
		})
		It("adds default labels", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			desiredSecondaryNodes := getSecondaryNodes()
			for id, object := range objectList {
				statefulSet := object.(*v1.StatefulSet)
				labels := statefulSet.Labels
				Expect(labels["app.kubernetes.io/name"]).To(Equal(Instance.Name))
				Expect(labels["app.kubernetes.io/component"]).To(Equal("teamcity-server"))
				Expect(labels["app.kubernetes.io/part-of"]).To(Equal("teamcity"))
				Expect(labels["teamcity.jetbrains.com/role"]).To(Equal("secondary"))
				Expect(labels["teamcity.jetbrains.com/node-name"]).To(Equal(desiredSecondaryNodes[id].Name))
			}

		})
		It("adds the correct label selector", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			desiredSecondaryNodes := getSecondaryNodes()
			for id, object := range objectList {
				statefulSet := object.(*v1.StatefulSet)
				labels := statefulSet.Spec.Selector.MatchLabels
				Expect(labels["app.kubernetes.io/name"]).To(Equal(Instance.Name))
				Expect(labels["app.kubernetes.io/component"]).To(Equal("teamcity-server"))
				Expect(labels["app.kubernetes.io/part-of"]).To(Equal("teamcity"))
				Expect(labels["teamcity.jetbrains.com/role"]).To(Equal("secondary"))
				Expect(labels["teamcity.jetbrains.com/node-name"]).To(Equal(desiredSecondaryNodes[id].Name))
			}
		})
		It("sets required resources requests for container", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			expected := v12.ResourceRequirements{
				Requests: v12.ResourceList{
					"cpu":    requests["cpu"],
					"memory": requests["memory"],
				},
				Limits: nil,
			}
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				Expect(statefulSet.Spec.Template.Spec.Containers[0].Resources).To(Equal(expected))
			}

		})
		It("sets prestop command for container", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				expected := []string{"/bin/sh", "-c", "/opt/teamcity/bin/shutdown.sh"}
				actual := statefulSet.Spec.Template.Spec.Containers[0].Lifecycle.PreStop.Exec.Command
				Expect(actual).To(Equal(expected))
			}
		})
		It("sets responsibility in serveropts correctly", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			secondaryNodes := getSecondaryNodes()
			for id, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				nodeResponsibilities := secondaryNodes[id].Spec.Responsibilities
				expectedResponsibility := ""
				for _, responsibility := range nodeResponsibilities {
					expectedResponsibility += fmt.Sprintf(" -Dteamcity.server.responsibilities=%s", responsibility)
				}
				envVars := statefulSet.Spec.Template.Spec.Containers[0].Env
				serverOptsIdx := slices.IndexFunc(envVars, func(v v12.EnvVar) bool { return v.Name == "TEAMCITY_SERVER_OPTS" })

				serverOpts := envVars[serverOptsIdx].Value

				Expect(strings.Contains(serverOpts, expectedResponsibility)).To(Equal(true))
			}
		})
		It("creates data dir volume and volume mount", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				Expect(len(statefulSet.Spec.Template.Spec.Volumes)).To(Equal(1))

				dataDirVolume := statefulSet.Spec.Template.Spec.Volumes[0]
				Expect(dataDirVolume.Name).To(Equal(dataDirPVC.Name))

				teamcityContainer := statefulSet.Spec.Template.Spec.Containers[0]
				Expect(len(teamcityContainer.VolumeMounts)).To(Equal(1))

				dataDirVolumeMount := teamcityContainer.VolumeMounts[0]
				Expect(dataDirVolumeMount.Name).To(Equal(dataDirPVC.VolumeMount.Name))
				Expect(dataDirVolumeMount.MountPath).To(Equal(dataDirPVC.VolumeMount.MountPath))
			}
		})
		It("returns obsolete objects correctly", func() {
			obsoleteObjects, err := DefaultSecondaryStatefulSetBuilder.GetObsoleteObjects(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(len(obsoleteObjects)).To(Equal(1))
			Expect(obsoleteObjects[0].GetName()).To(Equal(StaleStatefulSetName))
		})
	})
	Context("TeamCity with init containers", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Spec.SecondaryNodes[0].Spec.InitContainers = getInitContainers()
				teamcity.Spec.SecondaryNodes[1].Spec.InitContainers = getInitContainers()
			})
		})
		It("adds init containers", func() {
			initContainers := getInitContainers()
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				Expect(statefulSet.Spec.Template.Spec.InitContainers).To(Equal(initContainers))
			}
		})

	})
	Context("TeamCity with database properties", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Spec.DatabaseSecret = getDatabaseSecret()
			})

		})
		It("mounts database secret correctly", func() {

			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				envVars := statefulSet.Spec.Template.Spec.Containers[0].Env

				dbUserIdx := slices.IndexFunc(envVars, func(v v12.EnvVar) bool { return v.Name == "TEAMCITY_DB_USER" })
				Expect(envVars[dbUserIdx].ValueFrom.SecretKeyRef.LocalObjectReference).To(Equal(v12.LocalObjectReference{Name: Instance.Spec.DatabaseSecret.Secret}))
				Expect(envVars[dbUserIdx].ValueFrom.SecretKeyRef.Key).To(Equal("connectionProperties.user"))

				dbPasswordIdx := slices.IndexFunc(envVars, func(v v12.EnvVar) bool { return v.Name == "TEAMCITY_DB_PASSWORD" })
				Expect(envVars[dbPasswordIdx].ValueFrom.SecretKeyRef.LocalObjectReference).To(Equal(v12.LocalObjectReference{Name: Instance.Spec.DatabaseSecret.Secret}))
				Expect(envVars[dbPasswordIdx].ValueFrom.SecretKeyRef.Key).To(Equal("connectionProperties.password"))

				dbURLIdx := slices.IndexFunc(envVars, func(v v12.EnvVar) bool { return v.Name == "TEAMCITY_DB_URL" })
				Expect(envVars[dbURLIdx].ValueFrom.SecretKeyRef.LocalObjectReference).To(Equal(v12.LocalObjectReference{Name: Instance.Spec.DatabaseSecret.Secret}))
				Expect(envVars[dbURLIdx].ValueFrom.SecretKeyRef.Key).To(Equal("connectionUrl"))
			}

		})
	})
	Context("TeamCity with startup properties", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Spec.StartupPropertiesConfig = getStartupConfigurations()
			})
		})
		It("sets serveropts correctly", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				containerEnv := statefulSet.Spec.Template.Spec.Containers[0].Env
				serverOptsEnvVarIndex := slices.IndexFunc(containerEnv, func(c v12.EnvVar) bool { return c.Name == "TEAMCITY_SERVER_OPTS" })
				serverOpts := containerEnv[serverOptsEnvVarIndex].Value
				serverOptsSplit := strings.Fields(serverOpts)
				startupConfig := getStartupConfigurations()
				startUpServerOpts := serverOptsSplit[len(serverOptsSplit)-len(startupConfig):] //get elements of the split that correspond to startup vars

				keys := SortKeysAlphabeticallyInMap(startupConfig)
				i := 0

				for _, k := range keys {
					expectedValue := fmt.Sprintf("-D%s=%s", k, startupConfig[k])
					Expect(expectedValue).To(Equal(startUpServerOpts[i]))
					i += 1
				}
			}
		})
	})
	Context("TeamCity with additional mounts", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Spec.PersistentVolumeClaims = []CustomPersistentVolumeClaim{getAdditionalPVC()}
			})
		})
		It("sets additional volumes correctly", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				volumes := statefulSet.Spec.Template.Spec.Volumes
				Expect(len(volumes)).To(Equal(2))
			}
		})
	})
	Context("TeamCity with node selector", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Spec.SecondaryNodes[0].Spec.NodeSelector = getNodeSelector()
				teamcity.Spec.SecondaryNodes[1].Spec.NodeSelector = getNodeSelector()

			})
		})
		It("sets node selector terms correctly", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				Expect(statefulSet.Spec.Template.Spec.NodeSelector).To(Equal(getNodeSelector()))
			}
		})
	})
	Context("TeamCity with affinity", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Spec.SecondaryNodes[0].Spec.Affinity = getAffinity()
				teamcity.Spec.SecondaryNodes[1].Spec.Affinity = getAffinity()
			})
		})
		It("sets affinity correctly", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				affinity := statefulSet.Spec.Template.Spec.Affinity
				Expect(affinity.NodeAffinity).NotTo(Equal(nil))
				nodeSelectorTerms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
				Expect(len(nodeSelectorTerms)).To(Equal(1))
				Expect(nodeSelectorTerms[0].MatchExpressions[0].Key).To(Equal("some-key"))
				Expect(len(nodeSelectorTerms[0].MatchExpressions[0].Values)).To(Equal(1))
				Expect(nodeSelectorTerms[0].MatchExpressions[0].Values[0]).To(Equal("some-value"))
			}
		})
	})

	Context("TeamCity with custom labels", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.SecondaryNodes = getSecondaryNodes()
				teamcity.Labels = getLabels()
			})
		})
		It("sets labels correctly in StatefulSet spec", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				statefulSetLabels := statefulSet.Labels
				expectedLabels := getLabels()
				Expect(statefulSetLabels["foo"]).To(Equal(expectedLabels["foo"]))
				Expect(statefulSetLabels["teamcity"]).To(Equal(expectedLabels["teamcity"]))
			}

		})
		It("does not allow a label with existing key to override default label", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				statefulSetLabels := statefulSet.Labels
				providedLabels := getLabels()
				Expect(statefulSetLabels["app.kubernetes.io/name"]).ToNot(Equal(providedLabels["app.kubernetes.io/name"]))
			}
		})
	})

	Context("TeamCity with service account", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *TeamCity) {
				teamcity.Spec.ServiceAccount = getServiceAccount()
			})
		})
		It("sets serviceaccount name in StatefulSet spec", func() {
			objectList, err := DefaultSecondaryStatefulSetBuilder.BuildObjectList()
			Expect(err).NotTo(HaveOccurred())
			for _, object := range objectList {
				err = DefaultSecondaryStatefulSetBuilder.Update(object)
				statefulSet := object.(*v1.StatefulSet)
				serviceAccountName := statefulSet.Spec.Template.Spec.ServiceAccountName
				Expect(serviceAccountName).To(Equal(Instance.Spec.ServiceAccount.Name))
			}
		})
	})
})

type secondaryStatefulSetK8sClientMock struct {
	client.Client
}

func (m *secondaryStatefulSetK8sClientMock) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	listStatefulSet, ok := list.(*v1.StatefulSetList)
	if !ok {
		return fmt.Errorf("unable to convert object list to statefulset list")
	}
	secondaryNodes := getSecondaryNodes()

	listStatefulSet.Items = []v1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: secondaryNodes[0].Name,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: secondaryNodes[1].Name,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: StaleStatefulSetName,
			},
		},
	}
	return nil
}
