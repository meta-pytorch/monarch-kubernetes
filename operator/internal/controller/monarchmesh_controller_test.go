/*
BSD 3-Clause License

Copyright (c) Meta Platforms, Inc. and affiliates.
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monarchv1alpha1 "github.com/meta-pytorch/monarch-kubernetes/api/v1alpha1"
)

var _ = Describe("MonarchMesh Controller", func() {
	var (
		ctx                context.Context
		reconciler         *MonarchMeshReconciler
		config             Config
		typeNamespacedName types.NamespacedName
	)

	BeforeEach(func() {
		ctx = context.Background()
		config = DefaultConfig()
		reconciler = &MonarchMeshReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			Config: config,
		}
	})

	Context("When reconciling a non-existent resource", func() {
		It("should not return an error", func() {
			typeNamespacedName = types.NamespacedName{
				Name:      "non-existent-mesh",
				Namespace: "default",
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})

	Context("When reconciling a MonarchMesh resource", func() {
		const resourceName = "test-mesh"

		BeforeEach(func() {
			typeNamespacedName = types.NamespacedName{
				Name:      resourceName,
				Namespace: "default",
			}
		})

		AfterEach(func() {
			// Clean up the MonarchMesh resource
			mesh := &monarchv1alpha1.MonarchMesh{}
			err := k8sClient.Get(ctx, typeNamespacedName, mesh)
			if err == nil {
				Expect(k8sClient.Delete(ctx, mesh)).To(Succeed())
			}

			// Clean up the Service
			svc := &corev1.Service{}
			svcName := types.NamespacedName{
				Name:      resourceName + config.ServiceSuffix,
				Namespace: "default",
			}
			err = k8sClient.Get(ctx, svcName, svc)
			if err == nil {
				Expect(k8sClient.Delete(ctx, svc)).To(Succeed())
			}

			// Clean up the StatefulSet
			ss := &appsv1.StatefulSet{}
			err = k8sClient.Get(ctx, typeNamespacedName, ss)
			if err == nil {
				Expect(k8sClient.Delete(ctx, ss)).To(Succeed())
			}
		})

		It("should create a headless Service with correct configuration", func() {
			By("Creating the MonarchMesh resource")
			mesh := &monarchv1alpha1.MonarchMesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: monarchv1alpha1.MonarchMeshSpec{
					Replicas: 3,
					PodTemplate: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "worker",
							Image: "monarch:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mesh)).To(Succeed())

			By("Reconciling the resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Service was created correctly")
			svc := &corev1.Service{}
			svcName := types.NamespacedName{
				Name:      resourceName + config.ServiceSuffix,
				Namespace: "default",
			}
			Expect(k8sClient.Get(ctx, svcName, svc)).To(Succeed())

			// Verify headless service
			Expect(svc.Spec.ClusterIP).To(Equal("None"))

			// Verify labels
			Expect(svc.Labels).To(HaveKeyWithValue(config.MeshLabelKey, resourceName))
			Expect(svc.Labels).To(HaveKeyWithValue(config.AppLabelKey, config.AppLabelValue))

			// Verify selector
			Expect(svc.Spec.Selector).To(HaveKeyWithValue(config.MeshLabelKey, resourceName))
			Expect(svc.Spec.Selector).To(HaveKeyWithValue(config.AppLabelKey, config.AppLabelValue))

			// Verify port (should use default since not specified)
			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Name).To(Equal(config.PortName))
			Expect(svc.Spec.Ports[0].Port).To(Equal(config.DefaultPort))

			// Verify OwnerReference
			Expect(svc.OwnerReferences).To(HaveLen(1))
			Expect(svc.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(svc.OwnerReferences[0].Kind).To(Equal("MonarchMesh"))
		})

		It("should create a StatefulSet with correct configuration", func() {
			By("Creating the MonarchMesh resource")
			replicas := int32(5)
			mesh := &monarchv1alpha1.MonarchMesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: monarchv1alpha1.MonarchMeshSpec{
					Replicas: replicas,
					PodTemplate: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "worker",
							Image: "monarch:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mesh)).To(Succeed())

			By("Reconciling the resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the StatefulSet was created correctly")
			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, ss)).To(Succeed())

			// Verify replicas
			Expect(*ss.Spec.Replicas).To(Equal(replicas))

			// Verify service name
			Expect(ss.Spec.ServiceName).To(Equal(resourceName + config.ServiceSuffix))

			// Verify labels
			Expect(ss.Labels).To(HaveKeyWithValue(config.MeshLabelKey, resourceName))
			Expect(ss.Labels).To(HaveKeyWithValue(config.AppLabelKey, config.AppLabelValue))

			// Verify selector
			Expect(ss.Spec.Selector.MatchLabels).To(HaveKeyWithValue(config.MeshLabelKey, resourceName))
			Expect(ss.Spec.Selector.MatchLabels).To(HaveKeyWithValue(config.AppLabelKey, config.AppLabelValue))

			// Verify pod template labels
			Expect(ss.Spec.Template.Labels).To(HaveKeyWithValue(config.MeshLabelKey, resourceName))
			Expect(ss.Spec.Template.Labels).To(HaveKeyWithValue(config.AppLabelKey, config.AppLabelValue))

			// Verify parallel pod management policy
			Expect(ss.Spec.PodManagementPolicy).To(Equal(appsv1.ParallelPodManagement))

			// Verify pod template spec
			Expect(ss.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(ss.Spec.Template.Spec.Containers[0].Name).To(Equal("worker"))
			Expect(ss.Spec.Template.Spec.Containers[0].Image).To(Equal("monarch:latest"))

			// Verify OwnerReference
			Expect(ss.OwnerReferences).To(HaveLen(1))
			Expect(ss.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(ss.OwnerReferences[0].Kind).To(Equal("MonarchMesh"))
		})

		It("should use custom port when specified", func() {
			By("Creating the MonarchMesh resource with custom port")
			customPort := int32(12345)
			mesh := &monarchv1alpha1.MonarchMesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: monarchv1alpha1.MonarchMeshSpec{
					Replicas: 2,
					Port:     customPort,
					PodTemplate: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "worker",
							Image: "monarch:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mesh)).To(Succeed())

			By("Reconciling the resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Service uses the custom port")
			svc := &corev1.Service{}
			svcName := types.NamespacedName{
				Name:      resourceName + config.ServiceSuffix,
				Namespace: "default",
			}
			Expect(k8sClient.Get(ctx, svcName, svc)).To(Succeed())

			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(customPort))
		})

		It("should update MonarchMesh status after reconciliation", func() {
			By("Creating the MonarchMesh resource")
			mesh := &monarchv1alpha1.MonarchMesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: monarchv1alpha1.MonarchMeshSpec{
					Replicas: 3,
					PodTemplate: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "worker",
							Image: "monarch:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mesh)).To(Succeed())

			By("Reconciling the resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the status was updated")
			updatedMesh := &monarchv1alpha1.MonarchMesh{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedMesh)).To(Succeed())

			// Status should show condition (initially not ready since StatefulSet just created)
			Expect(updatedMesh.Status.Conditions).NotTo(BeEmpty())

			readyCondition := meta.FindStatusCondition(updatedMesh.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			// StatefulSet has no ready replicas yet, so Ready should be False
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("Waiting"))
		})

		It("should update existing resources on re-reconciliation", func() {
			By("Creating the MonarchMesh resource with initial replicas")
			mesh := &monarchv1alpha1.MonarchMesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: monarchv1alpha1.MonarchMeshSpec{
					Replicas: 2,
					PodTemplate: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "worker",
							Image: "monarch:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mesh)).To(Succeed())

			By("Reconciling the resource initially")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying initial StatefulSet replicas")
			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, ss)).To(Succeed())
			Expect(*ss.Spec.Replicas).To(Equal(int32(2)))

			By("Updating the MonarchMesh replicas")
			Expect(k8sClient.Get(ctx, typeNamespacedName, mesh)).To(Succeed())
			mesh.Spec.Replicas = 5
			Expect(k8sClient.Update(ctx, mesh)).To(Succeed())

			By("Reconciling again")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying StatefulSet replicas were updated")
			Expect(k8sClient.Get(ctx, typeNamespacedName, ss)).To(Succeed())
			Expect(*ss.Spec.Replicas).To(Equal(int32(5)))
		})
	})

	Context("When MonarchMesh is deleted", func() {
		It("should handle deletion gracefully", func() {
			const resourceName = "delete-test-mesh"
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: "default",
			}

			By("Creating and reconciling the MonarchMesh resource")
			mesh := &monarchv1alpha1.MonarchMesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: monarchv1alpha1.MonarchMeshSpec{
					Replicas: 1,
					PodTemplate: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "worker",
							Image: "monarch:latest",
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mesh)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying resources were created")
			ss := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, ss)).To(Succeed())

			svc := &corev1.Service{}
			svcName := types.NamespacedName{
				Name:      resourceName + config.ServiceSuffix,
				Namespace: "default",
			}
			Expect(k8sClient.Get(ctx, svcName, svc)).To(Succeed())

			By("Deleting the MonarchMesh resource")
			Expect(k8sClient.Delete(ctx, mesh)).To(Succeed())

			By("Reconciling after deletion")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			By("Verifying MonarchMesh is deleted")
			deletedMesh := &monarchv1alpha1.MonarchMesh{}
			err = k8sClient.Get(ctx, typeNamespacedName, deletedMesh)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
