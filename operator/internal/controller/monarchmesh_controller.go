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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	monarchv1 "github.com/meta-pytorch/monarch-kubernetes/api/v1"
)

// MonarchMeshReconciler reconciles a MonarchMesh object
type MonarchMeshReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pytorch.monarch.io,resources=monarchmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pytorch.monarch.io,resources=monarchmeshes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pytorch.monarch.io,resources=monarchmeshes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MonarchMesh object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *MonarchMeshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Fetch the Mesh Object
	var mesh monarchv1.MonarchMesh
	if err := r.Get(ctx, req.NamespacedName, &mesh); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Define Identifiers
	labels := map[string]string{"monarch-mesh": mesh.Name, "app": "monarch-worker"}
	svcName := mesh.Name + "-svc"

	// 3. Ensure Headless Service using CreateOrUpdate
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: mesh.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec.ClusterIP = "None"
		svc.Spec.Selector = labels
		svc.Spec.Ports = []corev1.ServicePort{{Name: "monarch", Port: 26600}}
		return ctrl.SetControllerReference(&mesh, svc, r.Scheme)
	})
	if err != nil {
		log.Error(err, "Failed to create or update Service")
		return ctrl.Result{}, err
	}

	// 4. Ensure StatefulSet using CreateOrUpdate
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: mesh.Name, Namespace: mesh.Namespace},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, ss, func() error {
		ss.Labels = labels
		ss.Spec.Replicas = mesh.Spec.Replicas
		ss.Spec.ServiceName = svcName
		ss.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		ss.Spec.Template.ObjectMeta.Labels = labels
		ss.Spec.Template.Spec = mesh.Spec.Template.Spec
		return ctrl.SetControllerReference(&mesh, ss, r.Scheme)
	})
	if err != nil {
		log.Error(err, "Failed to create or update StatefulSet")
		return ctrl.Result{}, err
	}

	// 5. Update Status
	mesh.Status.Replicas = ss.Status.Replicas
	mesh.Status.ReadyReplicas = ss.Status.ReadyReplicas
	mesh.Status.ServiceName = fmt.Sprintf("%s.%s.svc.cluster.local", svcName, mesh.Namespace)

	condition := metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Waiting"}
	if mesh.Spec.Replicas != nil && ss.Status.ReadyReplicas == *mesh.Spec.Replicas {
		condition = metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "AllReady"}
	}
	meta.SetStatusCondition(&mesh.Status.Conditions, condition)

	if err := r.Status().Update(ctx, &mesh); err != nil {
		log.Error(err, "Failed to update MonarchMesh status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MonarchMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monarchv1.MonarchMesh{}).
		// Reconcile on changes to the StatefulSet.
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
