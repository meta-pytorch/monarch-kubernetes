/*
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Package webhook contains startup bootstrap helpers for the conversion webhook.
//
// RBAC requirements for the cert bootstrap:
//   - Get/Create the cert Secret in the operator's namespace.
//   - Get/Update the MonarchMesh CRD to patch the caBundle.
//
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;update,resourceNames=monarchmeshes.monarch.pytorch.org
// The operator generates a self-signed CA + serving cert on first start (or reads
// them back from a Secret on restart), writes the serving cert to a directory the
// webhook server reads, and patches the CRD's spec.conversion.webhook.clientConfig.caBundle
// so the kube-apiserver trusts the cert when it calls /convert.
//
// This avoids the runtime dependency on cert-manager.
package webhook

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertBootstrapOptions configures EnsureCerts.
type CertBootstrapOptions struct {
	// Namespace is the namespace the operator runs in (used for the Secret and
	// the webhook Service DNS SANs).
	Namespace string
	// ServiceName is the name of the webhook Service the apiserver will dial.
	ServiceName string
	// SecretName is the name of the Secret used to persist the CA + serving cert
	// across operator restarts.
	SecretName string
	// CertDir is where tls.crt and tls.key will be written for the webhook server.
	CertDir string
	// CRDName is the CRD whose spec.conversion.webhook.clientConfig.caBundle should
	// be patched to trust the freshly issued CA.
	CRDName string
	// CertValidity is the lifetime of the serving cert. Defaults to 1 year.
	CertValidity time.Duration
	// CAValidity is the lifetime of the CA. Defaults to 10 years.
	CAValidity time.Duration
}

// EnsureCerts provisions (or loads) the webhook CA + serving cert, writes the
// serving cert to disk, and patches the CRD's caBundle. Safe to call on every
// operator startup — when the Secret already exists, the cert is reused.
func EnsureCerts(ctx context.Context, c client.Client, opts CertBootstrapOptions) error {
	if opts.CertValidity == 0 {
		opts.CertValidity = 365 * 24 * time.Hour
	}
	if opts.CAValidity == 0 {
		opts.CAValidity = 10 * 365 * 24 * time.Hour
	}

	ca, cert, key, err := loadOrCreateCerts(ctx, c, opts)
	if err != nil {
		return fmt.Errorf("provision webhook certs: %w", err)
	}

	if err := writeServingCert(opts.CertDir, cert, key); err != nil {
		return fmt.Errorf("write serving cert: %w", err)
	}

	if err := patchCRDCABundle(ctx, c, opts.CRDName, ca); err != nil {
		return fmt.Errorf("patch CRD caBundle: %w", err)
	}
	return nil
}

func loadOrCreateCerts(ctx context.Context, c client.Client, opts CertBootstrapOptions) (caPEM, certPEM, keyPEM []byte, err error) {
	key := client.ObjectKey{Namespace: opts.Namespace, Name: opts.SecretName}
	secret := &corev1.Secret{}
	if err := c.Get(ctx, key, secret); err == nil {
		ca := secret.Data["ca.crt"]
		cert := secret.Data["tls.crt"]
		serverKey := secret.Data["tls.key"]
		if len(ca) > 0 && len(cert) > 0 && len(serverKey) > 0 {
			return ca, cert, serverKey, nil
		}
		// Fall through: regenerate and overwrite.
	} else if !apierrors.IsNotFound(err) {
		return nil, nil, nil, err
	}

	caPEM, certPEM, keyPEM, err = generateCerts(opts)
	if err != nil {
		return nil, nil, nil, err
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      opts.SecretName,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":  caPEM,
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}
	if err := c.Create(ctx, newSecret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, nil, nil, err
		}
		// Lost the race against another replica. Re-read whatever they wrote.
		existing := &corev1.Secret{}
		if err := c.Get(ctx, key, existing); err != nil {
			return nil, nil, nil, err
		}
		return existing.Data["ca.crt"], existing.Data["tls.crt"], existing.Data["tls.key"], nil
	}
	return caPEM, certPEM, keyPEM, nil
}

func generateCerts(opts CertBootstrapOptions) (caPEM, certPEM, keyPEM []byte, err error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate CA key: %w", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "monarch-operator-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(opts.CAValidity),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("self-sign CA: %w", err)
	}
	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	servKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate serving key: %w", err)
	}
	dns := []string{
		fmt.Sprintf("%s.%s.svc", opts.ServiceName, opts.Namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", opts.ServiceName, opts.Namespace),
	}
	servTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano() + 1),
		Subject:      pkix.Name{CommonName: dns[0]},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(opts.CertValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dns,
	}
	parsedCA, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, nil, nil, err
	}
	servDER, err := x509.CreateCertificate(rand.Reader, servTmpl, parsedCA, &servKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("sign serving cert: %w", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servDER})

	keyDER, err := x509.MarshalECPrivateKey(servKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal serving key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return caPEM, certPEM, keyPEM, nil
}

func writeServingCert(dir string, cert, key []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tls.crt"), cert, 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "tls.key"), key, 0o600)
}

func patchCRDCABundle(ctx context.Context, c client.Client, crdName string, caBundle []byte) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := c.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		return err
	}
	if crd.Spec.Conversion == nil ||
		crd.Spec.Conversion.Strategy != apiextensionsv1.WebhookConverter ||
		crd.Spec.Conversion.Webhook == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig == nil {
		return fmt.Errorf(
			"CRD %s does not declare spec.conversion.strategy=Webhook with a clientConfig; "+
				"ensure the CRD manifest is installed with the conversion stanza",
			crdName,
		)
	}
	crd.Spec.Conversion.Webhook.ClientConfig.CABundle = caBundle
	return c.Update(ctx, crd)
}
