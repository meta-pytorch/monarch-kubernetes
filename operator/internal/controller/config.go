/*
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 */

package controller

// Config holds configuration for the MonarchMesh controller.
// These values can be overridden via controller flags in a future iteration.
type Config struct {
	// MeshLabelKey is the FQDN label key used to identify MonarchMesh-owned resources.
	// The value of this label will be set to the MonarchMesh name.
	// Uses FQDN convention to avoid collisions per:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	MeshLabelKey string

	// AppLabelKey is the standard Kubernetes label key for application name.
	AppLabelKey string

	// AppLabelValue is the value for the app label on pods.
	// This is used for pod selection and identification.
	AppLabelValue string

	// DefaultPort is the default port for Monarch mesh communication
	// when not specified in the MonarchMesh spec.
	DefaultPort int32

	// ServiceSuffix is appended to the MonarchMesh name to form the headless service name.
	ServiceSuffix string

	// PortName is the name used for the service port.
	PortName string
}

// DefaultConfig returns the default controller configuration.
// These defaults are suitable for most Monarch deployments.
func DefaultConfig() Config {
	return Config{
		MeshLabelKey:  "monarch.pytorch.org/mesh-name",
		AppLabelKey:   "app.kubernetes.io/name",
		AppLabelValue: "monarch-worker",
		DefaultPort:   26600,
		ServiceSuffix: "-svc",
		PortName:      "monarch",
	}
}
