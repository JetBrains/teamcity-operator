package controller

import (
	"context"
	"git.jetbrains.team/tch/teamcity-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var _ = Describe("TeamCity controller", func() {
	const (
		TeamCityName      = "test"
		TeamCityNamespace = "default"
		TeamCityImage     = "jetbrains/teamcity-server:latest"
	)

	Context("In the beginning", func() {
		It("Should be able to create new TeamCity", func() {
			By("By creating a new TeamCity resource")
			ctx := context.Background()
			object := &v1alpha1.TeamCity{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "jetbrains.com/v1alpha1",
					Kind:       "TeamCity",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      TeamCityName,
					Namespace: TeamCityNamespace,
				},
				Spec: v1alpha1.TeamCitySpec{
					Image: TeamCityImage,
				},
			}
			time.Sleep(10 * time.Second)
			Expect(k8sClient.Create(ctx, object)).Should(Succeed())
		})

	})
})
