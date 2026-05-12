/*
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 */

package v1alpha1

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monarchv1alpha2 "github.com/meta-pytorch/monarch-kubernetes/api/v1alpha2"
)

func TestConvertTo_WrapsPodSpecIntoPodTemplateSpec(t *testing.T) {
	src := &MonarchMesh{
		ObjectMeta: metav1.ObjectMeta{Name: "mesh-1", Namespace: "default"},
		Spec: MonarchMeshSpec{
			Replicas: 3,
			Port:     26600,
			PodTemplate: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "worker", Image: "monarch:v1"}},
			},
		},
		Status: MonarchMeshStatus{Replicas: 3, ReadyReplicas: 2},
	}

	dst := &monarchv1alpha2.MonarchMesh{}
	if err := src.ConvertTo(dst); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}

	if dst.Name != "mesh-1" {
		t.Errorf("Name: got %q, want %q", dst.Name, "mesh-1")
	}
	if dst.Spec.Replicas != 3 {
		t.Errorf("Replicas: got %d, want 3", dst.Spec.Replicas)
	}
	if dst.Spec.Port != 26600 {
		t.Errorf("Port: got %d, want 26600", dst.Spec.Port)
	}
	if got := len(dst.Spec.PodTemplate.Spec.Containers); got != 1 {
		t.Fatalf("Containers: got %d, want 1", got)
	}
	if got := dst.Spec.PodTemplate.Spec.Containers[0].Image; got != "monarch:v1" {
		t.Errorf("Image: got %q, want %q", got, "monarch:v1")
	}
	// v1alpha1 has no pod-level metadata, so this must remain empty after conversion.
	if len(dst.Spec.PodTemplate.Labels) != 0 {
		t.Errorf("PodTemplate.Labels should be empty, got %v", dst.Spec.PodTemplate.Labels)
	}
	if len(dst.Spec.PodTemplate.Annotations) != 0 {
		t.Errorf("PodTemplate.Annotations should be empty, got %v", dst.Spec.PodTemplate.Annotations)
	}
	if dst.Status.ReadyReplicas != 2 {
		t.Errorf("ReadyReplicas: got %d, want 2", dst.Status.ReadyReplicas)
	}
}

func TestConvertFrom_DropsPodTemplateMetadata(t *testing.T) {
	src := &monarchv1alpha2.MonarchMesh{
		ObjectMeta: metav1.ObjectMeta{Name: "mesh-2", Namespace: "default"},
		Spec: monarchv1alpha2.MonarchMeshSpec{
			Replicas: 2,
			PodTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"team": "monarch"},
					Annotations: map[string]string{"sidecar.istio.io/inject": "false"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "worker", Image: "monarch:v2"}},
				},
			},
		},
	}

	dst := &MonarchMesh{}
	if err := dst.ConvertFrom(src); err != nil {
		t.Fatalf("ConvertFrom: %v", err)
	}

	if dst.Name != "mesh-2" {
		t.Errorf("Name: got %q, want %q", dst.Name, "mesh-2")
	}
	if dst.Spec.Replicas != 2 {
		t.Errorf("Replicas: got %d, want 2", dst.Spec.Replicas)
	}
	if got := len(dst.Spec.PodTemplate.Containers); got != 1 {
		t.Fatalf("Containers: got %d, want 1", got)
	}
	if got := dst.Spec.PodTemplate.Containers[0].Image; got != "monarch:v2" {
		t.Errorf("Image: got %q, want %q", got, "monarch:v2")
	}
}

func TestRoundTrip_v1alpha1Preserved(t *testing.T) {
	original := &MonarchMesh{
		ObjectMeta: metav1.ObjectMeta{Name: "mesh-3"},
		Spec: MonarchMeshSpec{
			Replicas: 5,
			Port:     12345,
			PodTemplate: corev1.PodSpec{
				ServiceAccountName: "monarch-sa",
				Containers: []corev1.Container{{
					Name:    "worker",
					Image:   "monarch:v3",
					Command: []string{"/bin/sleep", "infinity"},
				}},
			},
		},
	}

	hub := &monarchv1alpha2.MonarchMesh{}
	if err := original.ConvertTo(hub); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}

	roundTripped := &MonarchMesh{}
	if err := roundTripped.ConvertFrom(hub); err != nil {
		t.Fatalf("ConvertFrom: %v", err)
	}

	if !reflect.DeepEqual(original.Spec, roundTripped.Spec) {
		t.Errorf("round-trip spec mismatch:\noriginal: %+v\ngot:      %+v", original.Spec, roundTripped.Spec)
	}
}
