/*
 Copyright 2021 The Hybridnet Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package networking_test

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	networkingv1 "github.com/alibaba/hybridnet/pkg/apis/networking/v1"
	"github.com/alibaba/hybridnet/pkg/constants"
	"github.com/alibaba/hybridnet/pkg/controllers/utils"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("Pod controller integration test suite", func() {
	Context("Lock", func() {
		testLock.Lock()
	})

	Context("IP allocation for single pod", func() {
		var podName string

		BeforeEach(func() {
			podName = fmt.Sprintf("pod-%s", uuid.NewUUID())
		})

		It("Allocate IPv4 address of underlay network for single pod", func() {
			By("create single pod on a node who has underlay network")
			pod := simplePodRender(podName, node1Name)
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			By("check IPv4 address allocation")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: "Pod",
						Name: pod.Name,
						UID:  pod.UID,
					}))

					g.Expect(ipInstance.Spec.Network).To(Equal(underlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(underlaySubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove the test pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())
		})

		It("Allocate IPv6 address of overlay network for single pod", func() {
			By("create single pod requiring overlay network and IPv6 address")
			pod := simplePodRender(podName, node3Name)
			pod.Annotations = map[string]string{
				constants.AnnotationNetworkType: "Overlay",
				constants.AnnotationIPFamily:    "IPv6",
			}
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			By("check IPv6 address allocation")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: "Pod",
						Name: pod.Name,
						UID:  pod.UID,
					}))

					g.Expect(ipInstance.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove the test pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())
		})

		It("Allocate DualStack addresses of overlay network for single pod", func() {
			By("create a single pod requiring overlay network and DualStack addresses")
			pod := simplePodRender(podName, node3Name)
			pod.Annotations = map[string]string{
				constants.AnnotationNetworkType: "Overlay",
				constants.AnnotationIPFamily:    "DualStack",
			}
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			By("check DualStack addresses allocation")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(2))

					networkingv1.SortIPInstancePointerSlice(ipInstances)
					ipInstanceIPv4 := ipInstances[0]
					g.Expect(ipInstanceIPv4.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv4.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: "Pod",
						Name: pod.Name,
						UID:  pod.UID,
					}))

					g.Expect(ipInstanceIPv4.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv4.Spec.Subnet).To(BeElementOf(overlayIPv4SubnetName))

					ipInstanceIPv6 := ipInstances[1]
					g.Expect(ipInstanceIPv6.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv6.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: "Pod",
						Name: pod.Name,
						UID:  pod.UID,
					}))

					g.Expect(ipInstanceIPv6.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv6.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove the test pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(k8sClient.DeleteAllOf(
				context.Background(),
				&networkingv1.IPInstance{},
				client.MatchingLabels{
					constants.LabelPod: podName,
				},
				client.InNamespace("default"),
			))
		})
	})

	Context("IP retain for single stateful pod", func() {
		var podName string
		var ownerReference metav1.OwnerReference

		BeforeEach(func() {
			podName = fmt.Sprintf("pod-%d", rand.Intn(10))
			ownerReference = statefulOwnerReferenceRender()
		})

		It("Allocate and retain IPv4 address of underlay network for single stateful pod", func() {
			By("create a stateful pod requiring IPv4 address")
			var ipInstanceName string
			pod := simplePodRender(podName, node1Name)
			pod.OwnerReferences = []metav1.OwnerReference{ownerReference}
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			By("check the first allocated IPv4 address")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					ipInstanceName = ipInstance.Name
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.NodeName).To(Equal(node1Name))
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstance.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstance.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstance.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstance.Spec.Network).To(Equal(underlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(underlaySubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove stateful pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())

			By("check the allocated IPv4 address is reserved")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					g.Expect(ipInstance.Name).To(Equal(ipInstanceName))
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(BeEmpty())
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.NodeName).To(BeEmpty())
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstance.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstance.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstance.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstance.Spec.Network).To(Equal(underlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(underlaySubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			// TODO: check status in IPAM manager

			By("make sure pod deleted")
			Eventually(
				func(g Gomega) {
					err := k8sClient.Get(context.Background(),
						types.NamespacedName{
							Namespace: pod.Namespace,
							Name:      podName,
						},
						&corev1.Pod{})
					g.Expect(err).NotTo(BeNil())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("recreate the stateful pod")
			pod = simplePodRender(podName, node1Name)
			pod.OwnerReferences = []metav1.OwnerReference{ownerReference}
			Expect(k8sClient.Create(context.Background(), pod)).NotTo(HaveOccurred())

			By("check the allocated IPv4 address is retained and reused")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					g.Expect(ipInstance.Name).To(Equal(ipInstanceName))
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.NodeName).To(Equal(node1Name))
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstance.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstance.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstance.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstance.Spec.Network).To(Equal(underlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(underlaySubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove the test pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())
		})

		It("Allocate and retain IPv6 address of overlay network for single stateful pod", func() {
			By("create a stateful pod requiring IPv6 address and overlay network")
			var ipInstanceName string
			pod := simplePodRender(podName, node3Name)
			pod.OwnerReferences = []metav1.OwnerReference{ownerReference}
			pod.Annotations = map[string]string{
				constants.AnnotationNetworkType: "Overlay",
				constants.AnnotationIPFamily:    "IPv6",
			}
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			By("check the first allocated IPv6 address")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					ipInstanceName = ipInstance.Name
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.NodeName).To(Equal(node3Name))
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstance.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstance.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstance.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstance.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove stateful pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())

			By("check the allocated IPv6 address is reserved")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					g.Expect(ipInstance.Name).To(Equal(ipInstanceName))
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(BeEmpty())
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.NodeName).To(BeEmpty())
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstance.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstance.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstance.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstance.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			// TODO: check status in IPAM manager

			By("make sure pod deleted")
			Eventually(
				func(g Gomega) {
					err := k8sClient.Get(context.Background(),
						types.NamespacedName{
							Namespace: pod.Namespace,
							Name:      podName,
						},
						&corev1.Pod{})
					g.Expect(err).NotTo(BeNil())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("recreate the stateful pod on another node")
			pod = simplePodRender(podName, node1Name)
			pod.OwnerReferences = []metav1.OwnerReference{ownerReference}
			pod.Annotations = map[string]string{
				constants.AnnotationNetworkType: "Overlay",
				constants.AnnotationIPFamily:    "IPv6",
			}
			Expect(k8sClient.Create(context.Background(), pod)).NotTo(HaveOccurred())

			By("check the allocated IPv6 address is retained and reused")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(1))

					ipInstance := ipInstances[0]
					g.Expect(ipInstance.Name).To(Equal(ipInstanceName))
					g.Expect(ipInstance.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstance.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstance.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstance.Spec.Binding.NodeName).To(Equal(node1Name))
					g.Expect(ipInstance.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstance.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstance.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstance.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstance.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstance.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove the test pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())
		})

		It("Allocate and retain DualStack addresses of overlay network for single stateful pod", func() {
			By("create a stateful pod requiring DualStack addresses and overlay network")
			var ipInstanceIPv4Name string
			var ipInstanceIPv6Name string
			pod := simplePodRender(podName, node3Name)
			pod.OwnerReferences = []metav1.OwnerReference{ownerReference}
			pod.Annotations = map[string]string{
				constants.AnnotationNetworkType: "Overlay",
				constants.AnnotationIPFamily:    "DualStack",
			}
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			By("check the first allocated DualStack addresses")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(2))

					// sort by ip family order
					networkingv1.SortIPInstancePointerSlice(ipInstances)

					// check IPv4 IPInstance
					ipInstanceIPv4 := ipInstances[0]
					ipInstanceIPv4Name = ipInstanceIPv4.Name
					g.Expect(ipInstanceIPv4.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv4.Spec.Binding.NodeName).To(Equal(node3Name))
					g.Expect(ipInstanceIPv4.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstanceIPv4.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstanceIPv4.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstanceIPv4.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstanceIPv4.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv4.Spec.Subnet).To(BeElementOf(overlayIPv4SubnetName))

					// check IPv6 IPInstance
					ipInstanceIPv6 := ipInstances[1]
					ipInstanceIPv6Name = ipInstanceIPv6.Name
					g.Expect(ipInstanceIPv6.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv6.Spec.Binding.NodeName).To(Equal(node3Name))
					g.Expect(ipInstanceIPv6.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstanceIPv6.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstanceIPv6.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx = *ipInstanceIPv6.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstanceIPv6.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv6.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))

					// check MAC address
					g.Expect(ipInstanceIPv4.Spec.Address.MAC).To(Equal(ipInstanceIPv6.Spec.Address.MAC))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove stateful pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())

			By("check the allocated DualStack addresses are reserved")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(2))

					// sort by ip family order
					networkingv1.SortIPInstancePointerSlice(ipInstances)

					// check IPv4 IPInstance
					ipInstanceIPv4 := ipInstances[0]
					g.Expect(ipInstanceIPv4.Name).To(Equal(ipInstanceIPv4Name))
					g.Expect(ipInstanceIPv4.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodUID).To(BeEmpty())
					g.Expect(ipInstanceIPv4.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv4.Spec.Binding.NodeName).To(BeEmpty())
					g.Expect(ipInstanceIPv4.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstanceIPv4.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstanceIPv4.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstanceIPv4.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstanceIPv4.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv4.Spec.Subnet).To(BeElementOf(overlayIPv4SubnetName))

					// check IPv6 IPInstance
					ipInstanceIPv6 := ipInstances[1]
					g.Expect(ipInstanceIPv6.Name).To(Equal(ipInstanceIPv6Name))
					g.Expect(ipInstanceIPv6.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodUID).To(BeEmpty())
					g.Expect(ipInstanceIPv6.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv6.Spec.Binding.NodeName).To(BeEmpty())
					g.Expect(ipInstanceIPv6.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstanceIPv6.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstanceIPv6.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx = *ipInstanceIPv6.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstanceIPv6.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv6.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))

					// check MAC address
					g.Expect(ipInstanceIPv4.Spec.Address.MAC).To(Equal(ipInstanceIPv6.Spec.Address.MAC))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			// TODO: check status in IPAM manager

			By("make sure pod deleted")
			Eventually(
				func(g Gomega) {
					err := k8sClient.Get(context.Background(),
						types.NamespacedName{
							Namespace: pod.Namespace,
							Name:      podName,
						},
						&corev1.Pod{})
					g.Expect(err).NotTo(BeNil())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("recreate the stateful pod on another node")
			pod = simplePodRender(podName, node1Name)
			pod.OwnerReferences = []metav1.OwnerReference{ownerReference}
			pod.Annotations = map[string]string{
				constants.AnnotationNetworkType: "Overlay",
				constants.AnnotationIPFamily:    "DualStack",
			}
			Expect(k8sClient.Create(context.Background(), pod)).NotTo(HaveOccurred())

			By("check the allocated DualStack addresses are retained and reused")
			Eventually(
				func(g Gomega) {
					ipInstances, err := utils.ListAllocatedIPInstancesOfPod(context.Background(), k8sClient, pod)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ipInstances).To(HaveLen(2))

					// sort by ip family order
					networkingv1.SortIPInstancePointerSlice(ipInstances)

					// check IPv4 IPInstance
					ipInstanceIPv4 := ipInstances[0]
					g.Expect(ipInstanceIPv4.Name).To(Equal(ipInstanceIPv4Name))
					g.Expect(ipInstanceIPv4.Spec.Address.Version).To(Equal(networkingv1.IPv4))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstanceIPv4.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv4.Spec.Binding.NodeName).To(Equal(node1Name))
					g.Expect(ipInstanceIPv4.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstanceIPv4.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstanceIPv4.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx := *ipInstanceIPv4.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstanceIPv4.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv4.Spec.Subnet).To(BeElementOf(overlayIPv4SubnetName))

					// check IPv6 IPInstance
					ipInstanceIPv6 := ipInstances[1]
					g.Expect(ipInstanceIPv6.Name).To(Equal(ipInstanceIPv6Name))
					g.Expect(ipInstanceIPv6.Spec.Address.Version).To(Equal(networkingv1.IPv6))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodUID).To(Equal(pod.UID))
					g.Expect(ipInstanceIPv6.Spec.Binding.PodName).To(Equal(pod.Name))
					g.Expect(ipInstanceIPv6.Spec.Binding.NodeName).To(Equal(node1Name))
					g.Expect(ipInstanceIPv6.Spec.Binding.ReferredObject).To(Equal(networkingv1.ObjectMeta{
						Kind: ownerReference.Kind,
						Name: ownerReference.Name,
						UID:  ownerReference.UID,
					}))

					g.Expect(ipInstanceIPv6.Spec.Binding.Stateful).NotTo(BeNil())
					g.Expect(ipInstanceIPv6.Spec.Binding.Stateful.Index).NotTo(BeNil())

					idx = *ipInstanceIPv6.Spec.Binding.Stateful.Index
					g.Expect(pod.Name).To(Equal(fmt.Sprintf("pod-%d", idx)))

					g.Expect(ipInstanceIPv6.Spec.Network).To(Equal(overlayNetworkName))
					g.Expect(ipInstanceIPv6.Spec.Subnet).To(BeElementOf(overlayIPv6SubnetName))

					// check MAC address
					g.Expect(ipInstanceIPv4.Spec.Address.MAC).To(Equal(ipInstanceIPv6.Spec.Address.MAC))
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())

			By("remove the test pod")
			Expect(k8sClient.Delete(context.Background(), pod, client.GracePeriodSeconds(0))).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("make sure test ip instances cleaned up")
			Expect(k8sClient.DeleteAllOf(
				context.Background(),
				&networkingv1.IPInstance{},
				client.MatchingLabels{
					constants.LabelPod: podName,
				},
				client.InNamespace("default"),
			))

			By("make sure test pod cleaned up")
			Eventually(
				func(g Gomega) {
					err := k8sClient.Get(context.Background(),
						types.NamespacedName{
							Namespace: "default",
							Name:      podName,
						},
						&corev1.Pod{})
					g.Expect(err).NotTo(BeNil())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).
				WithTimeout(30 * time.Second).
				WithPolling(time.Second).
				Should(Succeed())
		})
	})

	Context("Unlock", func() {
		testLock.Unlock()
	})
})

func simplePodRender(name string, node string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: node,
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "test",
				},
			},
		},
	}
}

func statefulOwnerReferenceRender() metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "StatefulSet",
		Name:               "fake",
		UID:                uuid.NewUUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}