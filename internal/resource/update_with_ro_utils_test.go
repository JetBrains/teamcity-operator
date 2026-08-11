package resource

import (
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("UpdateWithROUtils", func() {
	Context("BuildRoNode", func() {
		It("creates a node with the correct name and resources from main node", func() {
			instance := &TeamCity{
				Spec: TeamCitySpec{
					MainNode: Node{
						Name: "main-node",
						Spec: NodeSpec{
							Requests: v12.ResourceList{
								"cpu":    resource.MustParse("1000m"),
								"memory": resource.MustParse("2Gi"),
							},
						},
					},
				},
			}

			roNode := BuildRoNode(instance, "main-node-update-replica")

			Expect(roNode.Name).To(Equal("main-node-update-replica"))
			Expect(roNode.Spec.Requests["cpu"]).To(Equal(resource.MustParse("1000m")))
			Expect(roNode.Spec.Requests["memory"]).To(Equal(resource.MustParse("2Gi")))
		})
	})

	Context("GetROStatefulSetNamespacedName", func() {
		It("returns the correct namespaced name with postfix", func() {
			instance := &TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tc",
					Namespace: "test-namespace",
				},
				Spec: TeamCitySpec{
					MainNode: Node{
						Name: "main-node",
					},
				},
			}

			result := GetROStatefulSetNamespacedName(instance)

			Expect(result.Name).To(Equal("main-node-update-replica"))
			Expect(result.Namespace).To(Equal("test-namespace"))
		})
	})

	Context("BuildROStatefulSet", func() {
		It("creates a StatefulSet with correct name and labels", func() {
			instance := &TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tc",
					Namespace: "test-namespace",
				},
				Spec: TeamCitySpec{
					MainNode: Node{
						Name: "main-node",
						Spec: NodeSpec{
							Requests: v12.ResourceList{
								"cpu":    resource.MustParse("500m"),
								"memory": resource.MustParse("1Gi"),
							},
						},
					},
				},
			}

			roStatefulSet := BuildROStatefulSet(instance)

			Expect(roStatefulSet.Name).To(Equal("main-node-update-replica"))
			Expect(roStatefulSet.Namespace).To(Equal("test-namespace"))
			Expect(roStatefulSet.Labels["teamcity.jetbrains.com/role"]).To(Equal(RoNodeRole))
		})
	})

	Context("UpdateROStatefulSet with node data dir", func() {
		It("replaces node-data PVC with emptyDir and keeps JVM property", func() {
			scheme := runtime.NewScheme()
			Expect(AddToScheme(scheme)).To(Succeed())

			instance := &TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tc",
					Namespace: "test-namespace",
					UID:       "uid-1",
				},
				Spec: TeamCitySpec{
					Image:         "jetbrains/teamcity-server:latest",
					XmxPercentage: 95,
					DataDirVolumeClaim: CustomPersistentVolumeClaim{
						Name: "data-dir",
						VolumeMount: v12.VolumeMount{
							Name:      "data",
							MountPath: "/data/teamcity",
						},
					},
					MainNode: Node{
						Name: "main-node",
						Spec: NodeSpec{
							Requests: v12.ResourceList{
								"cpu":    resource.MustParse("500m"),
								"memory": resource.MustParse("1Gi"),
							},
							NodeDataDirVolumeClaim: &CustomPersistentVolumeClaim{
								Name: "node-data-dir-main-node",
								VolumeMount: v12.VolumeMount{
									Name:      "node-data-dir",
									MountPath: "/mnt/node-data-dir",
								},
								Spec: v12.PersistentVolumeClaimSpec{
									Resources: v12.ResourceRequirements{
										Requests: v12.ResourceList{
											v12.ResourceStorage: resource.MustParse("1Gi"),
										},
									},
								},
							},
						},
					},
				},
			}

			main := &v1.StatefulSet{}
			ConfigureStatefulSet(instance, instance.Spec.MainNode, main)
			var container v12.Container
			ConfigureContainer(instance, instance.Spec.MainNode, &container)
			main.Spec.Template.Spec.Containers = []v12.Container{container}

			ro := BuildROStatefulSet(instance)
			Expect(UpdateROStatefulSet(scheme, instance, main, ro)).To(Succeed())

			var nodeDataVolume *v12.Volume
			for i := range ro.Spec.Template.Spec.Volumes {
				if ro.Spec.Template.Spec.Volumes[i].Name == "node-data-dir" {
					nodeDataVolume = &ro.Spec.Template.Spec.Volumes[i]
					break
				}
			}
			Expect(nodeDataVolume).NotTo(BeNil())
			Expect(nodeDataVolume.EmptyDir).NotTo(BeNil())
			Expect(nodeDataVolume.PersistentVolumeClaim).To(BeNil())

			env := ro.Spec.Template.Spec.Containers[0].Env
			var serverOpts string
			for _, e := range env {
				if e.Name == "TEAMCITY_SERVER_OPTS" {
					serverOpts = e.Value
				}
			}
			Expect(serverOpts).To(ContainSubstring("-Dteamcity.node.data.path=/mnt/node-data-dir"))
		})
	})

	Context("ChangesRequireNodeStatefulSetRestart", func() {
		var instance *TeamCity
		var node Node

		BeforeEach(func() {
			instance = &TeamCity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tc",
					Namespace: "test-namespace",
				},
				Spec: TeamCitySpec{
					Image: "jetbrains/teamcity-server:latest",
					DataDirVolumeClaim: CustomPersistentVolumeClaim{
						Name: "data-dir",
						VolumeMount: v12.VolumeMount{
							Name:      "data",
							MountPath: "/data/teamcity",
						},
					},
					MainNode: Node{
						Name: "main-node",
						Spec: NodeSpec{
							Requests: v12.ResourceList{
								"cpu":    resource.MustParse("500m"),
								"memory": resource.MustParse("1Gi"),
							},
						},
					},
				},
			}
			node = instance.Spec.MainNode
		})

		It("returns true when image changes", func() {
			existing := &v1.StatefulSet{
				Spec: v1.StatefulSetSpec{
					Template: v12.PodTemplateSpec{
						Spec: v12.PodSpec{
							Containers: []v12.Container{
								{
									Image: "jetbrains/teamcity-server:old",
								},
							},
						},
					},
				},
			}

			result := ChangesRequireNodeStatefulSetRestart(instance, node, existing)
			Expect(result).To(BeTrue())
		})

		It("returns true when resources change", func() {
			existing := &v1.StatefulSet{
				Spec: v1.StatefulSetSpec{
					Template: v12.PodTemplateSpec{
						Spec: v12.PodSpec{
							Containers: []v12.Container{
								{
									Image: "jetbrains/teamcity-server:latest",
									Resources: v12.ResourceRequirements{
										Requests: v12.ResourceList{
											"cpu":    resource.MustParse("200m"),
											"memory": resource.MustParse("512Mi"),
										},
									},
								},
							},
						},
					},
				},
			}

			result := ChangesRequireNodeStatefulSetRestart(instance, node, existing)
			Expect(result).To(BeTrue())
		})
	})
})
