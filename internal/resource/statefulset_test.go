package resource

import (
	"fmt"
	v1alpha1 "git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"strings"
)

var _ = Describe("StatefulSet", func() {
	Context("TeamCity with minimal configuration", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *v1alpha1.TeamCity) {})
		})
		It("sets a name and namespace", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			sts := obj.(*v1.StatefulSet)
			Expect(sts.Name).To(Equal(TeamCityName))
			Expect(sts.Namespace).To(Equal(TeamCityNamespace))
		})
		It("adds the correct label selector", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			labels := statefulSet.Spec.Selector.MatchLabels
			Expect(labels["app.kubernetes.io/name"]).To(Equal(Instance.Name))
		})
		It("sets required resources requests for container", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)
			expected := v12.ResourceRequirements{
				Requests: v12.ResourceList{
					"cpu":    builder.Instance.Spec.Requests["cpu"],
					"memory": builder.Instance.Spec.Requests["memory"],
				},
				Limits: nil,
			}
			actual := statefulSet.Spec.Template.Spec.Containers[0].Resources
			Expect(actual).To(Equal(expected))
		})
		It("sets prestop command for container", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)
			expected := []string{"/bin/sh", "-c", "/opt/teamcity/bin/shutdown.sh"}
			actual := statefulSet.Spec.Template.Spec.Containers[0].Lifecycle.PreStop.Exec.Command
			Expect(actual).To(Equal(expected))
		})
		It("calculates and provides env vars correctly", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)
			xmxValue := xmxValueCalculator(xmxPercentage, builder.Instance.Spec.Requests.Memory().Value())
			datadirPath := volumeMountsBuilder(builder.Instance)[0].MountPath

			dataPath := v12.EnvVar{
				Name:  "TEAMCITY_DATA_PATH",
				Value: datadirPath,
			}

			logsPath := v12.EnvVar{
				Name:  "TEAMCITY_LOGS_PATH",
				Value: fmt.Sprintf("%s/%s", datadirPath, "logs"),
			}

			memOpts := v12.EnvVar{
				Name:  "TEAMCITY_SERVER_MEM_OPTS",
				Value: fmt.Sprintf("%s%s", "-Xmx", xmxValue),
			}

			serverOpts := v12.EnvVar{
				Name: "TEAMCITY_SERVER_OPTS",
				Value: "-XX:+HeapDumpOnOutOfMemoryError -XX:+DisableExplicitGC" +
					fmt.Sprintf(" -XX:HeapDumpPath=%s%s%s", datadirPath, "/memoryDumps/", TeamCityName) +
					fmt.Sprintf(" -Dteamcity.server.nodeId=%s", TeamCityName) + fmt.Sprintf(" -Dteamcity.server.rootURL=%s", TeamCityName)}
			expected := append([]v12.EnvVar{}, memOpts, dataPath, logsPath, serverOpts)
			actual := statefulSet.Spec.Template.Spec.Containers[0].Env
			envVarsAreEqual := assert.ElementsMatch(GinkgoT(), expected, actual)
			Expect(envVarsAreEqual).To(Equal(true))
		})
		It("sets the owner reference", func() {
			statefulSet := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Instance.Name,
					Namespace: Instance.Namespace,
				},
			}
			Expect(DefaultStatefulSetBuilder.Update(statefulSet)).To(Succeed())
			Expect(len(statefulSet.OwnerReferences)).To(Equal(1))
			Expect(statefulSet.OwnerReferences[0].Name).To(Equal(builder.Instance.Name))
		})
		It("creates the required PersistentVolumeClaims", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			expected := []v12.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pvcName,
						Namespace: builder.Instance.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "jetbrains.com/v1alpha1",
								Kind:               "TeamCity",
								Name:               builder.Instance.GetName(),
								UID:                builder.Instance.GetUID(),
								BlockOwnerDeletion: pointer.Bool(true),
								Controller:         pointer.Bool(true),
							},
						},
					},
					Spec: pvcSpec,
				},
			}
			actual := statefulSet.Spec.VolumeClaimTemplates
			Expect(actual).To(Equal(expected))
		})
	})
	Context("TeamCity with init containers", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *v1alpha1.TeamCity) {
				teamcity.Spec.InitContainers = getInitContainers()
			})
		})
		It("adds init containers", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			sts := obj.(*v1.StatefulSet)
			Expect(sts.Spec.Template.Spec.InitContainers).To(Equal(getInitContainers()))
		})
		It("adds init containers after update", func() {
			statefulSet := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Instance.Name,
					Namespace: Instance.Namespace,
				},
			}
			Expect(DefaultStatefulSetBuilder.Update(statefulSet)).To(Succeed())
			Expect(len(statefulSet.OwnerReferences)).To(Equal(1))
			Expect(statefulSet.Spec.Template.Spec.InitContainers).To(Equal(getInitContainers()))
		})
	})
	Context("TeamCity with database properties", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *v1alpha1.TeamCity) {
				teamcity.Spec.DatabaseSecret = getDatabaseSecret()
			})

		})
		It("mounts database secret correctly", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

			volumes := statefulSet.Spec.Template.Spec.Volumes
			Expect(len(volumes)).To(Equal(1))

			databaseSecretVolume := volumes[0]
			Expect(databaseSecretVolume.Name).To(Equal(DATABASE_PROPERTIES_VOLUME_NAME))
			Expect(databaseSecretVolume.Secret.SecretName).To(Equal(Instance.Spec.DatabaseSecret.Secret))

			volumeMounts := statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts
			Expect(len(volumeMounts)).To(Equal(2))

			databaseSecretVolumeMount := volumeMounts[1]
			datadirPath := volumeMountsBuilder(builder.Instance)[0].MountPath
			datadirPathClean := strings.Replace(datadirPath, "/", "", -1)

			databaseSecretPathSplit := RemoveEmptyStrings(strings.Split(databaseSecretVolumeMount.MountPath, "/"))
			Expect(databaseSecretPathSplit).To(Equal([]string{datadirPathClean, "config", TEAMCITY_DATABASE_PROPERTIES_SUB_PATH}))
		})
	})
	Context("TeamCity with startup properties", func() {
		BeforeEach(func() {
			BeforeEachBuild(func(teamcity *v1alpha1.TeamCity) {
				teamcity.Spec.StartupPropertiesConfig = getStartupConfigurations()
			})
		})
		It("sets serveropts correctly", func() {
			obj, err := DefaultStatefulSetBuilder.Build()
			Expect(err).NotTo(HaveOccurred())
			err = DefaultStatefulSetBuilder.Update(obj)
			Expect(err).NotTo(HaveOccurred())
			statefulSet := obj.(*v1.StatefulSet)

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

		})
	})

})

func RemoveEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
