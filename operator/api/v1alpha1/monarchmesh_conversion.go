/*
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	monarchv1alpha2 "github.com/meta-pytorch/monarch-kubernetes/api/v1alpha2"
)

// ConvertTo converts this MonarchMesh (v1alpha1) to the Hub version (v1alpha2).
// v1alpha1 stores the pod template as a bare PodSpec, so we wrap it under PodTemplateSpec.Spec.
// v1alpha1 has no pod-level metadata, so PodTemplateSpec.ObjectMeta is left empty.
func (src *MonarchMesh) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*monarchv1alpha2.MonarchMesh)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Port = src.Spec.Port
	dst.Spec.PodTemplate = corev1.PodTemplateSpec{
		Spec: src.Spec.PodTemplate,
	}

	dst.Status.Replicas = src.Status.Replicas
	dst.Status.ReadyReplicas = src.Status.ReadyReplicas
	dst.Status.Conditions = src.Status.Conditions

	return nil
}

// ConvertFrom converts the Hub version (v1alpha2) back to this MonarchMesh (v1alpha1).
// Any pod-level metadata under PodTemplate.ObjectMeta is dropped because v1alpha1 has no
// way to represent it. Callers that need pod-level labels/annotations must use v1alpha2.
func (dst *MonarchMesh) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*monarchv1alpha2.MonarchMesh)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Port = src.Spec.Port
	dst.Spec.PodTemplate = src.Spec.PodTemplate.Spec

	dst.Status.Replicas = src.Status.Replicas
	dst.Status.ReadyReplicas = src.Status.ReadyReplicas
	dst.Status.Conditions = src.Status.Conditions

	return nil
}
