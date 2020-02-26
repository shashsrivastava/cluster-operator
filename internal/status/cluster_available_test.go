package status_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	rabbitmqstatus "github.com/pivotal/rabbitmq-for-kubernetes/internal/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("ClusterAvailable", func() {
	var (
		childServiceEndpoints *corev1.Endpoints
	)

	BeforeEach(func() {
		childServiceEndpoints = &corev1.Endpoints{}
	})

	When("at least one service endpoint is published", func() {
		BeforeEach(func() {
			childServiceEndpoints.Subsets = []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "1.2.3.4",
						},
						{
							IP: "5.6.7.8",
						},
					},
				},
			}
		})

		It("returns the expected condition", func() {
			condition := rabbitmqstatus.ClusterAvailableCondition([]runtime.Object{&corev1.Pod{}, childServiceEndpoints})
			By("having the correct type", func() {
				var conditionType rabbitmqstatus.RabbitmqClusterConditionType = "ClusterAvailable"
				Expect(condition.Type).To(Equal(conditionType))
			})

			By("having status true and reason message", func() {
				Expect(condition.Status).To(Equal(corev1.ConditionTrue))
				Expect(condition.Reason).To(Equal("AtLeastOneEndpointAvailable"))
			})
		})
	})

	When("no service endpoint is published", func() {
		BeforeEach(func() {
			childServiceEndpoints.Subsets = []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{},
				},
			}
		})

		It("returns the expected condition", func() {
			condition := rabbitmqstatus.ClusterAvailableCondition([]runtime.Object{childServiceEndpoints})
			By("having the correct type", func() {
				var conditionType rabbitmqstatus.RabbitmqClusterConditionType = "ClusterAvailable"
				Expect(condition.Type).To(Equal(conditionType))
			})

			By("having status true and reason message", func() {
				Expect(condition.Status).To(Equal(corev1.ConditionFalse))
				Expect(condition.Reason).To(Equal("NoEndpointsAvailable"))
				Expect(condition.Message).NotTo(BeEmpty())
			})
		})
	})

	When("service endpoints do not exist", func() {
		BeforeEach(func() {
			childServiceEndpoints = nil
		})

		It("returns the expected condition", func() {
			condition := rabbitmqstatus.ClusterAvailableCondition([]runtime.Object{childServiceEndpoints})
			By("having the correct type", func() {
				var conditionType rabbitmqstatus.RabbitmqClusterConditionType = "ClusterAvailable"
				Expect(condition.Type).To(Equal(conditionType))
			})

			By("having status true and reason message", func() {
				Expect(condition.Status).To(Equal(corev1.ConditionUnknown))
				Expect(condition.Reason).To(Equal("CouldNotRetrieveEndpoints"))
				Expect(condition.Message).NotTo(BeEmpty())
			})
		})
	})

})