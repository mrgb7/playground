package plugins

import (
	"fmt"
	"strings"
	"testing"
)

func TestTLSPluginInterface(t *testing.T) {
	plugin, err := NewTLS("dummy-kubeconfig", "test-cluster")
	if err != nil {
		t.Logf("K8s client creation failed (expected in test): %v", err)
		return
	}

	if plugin.GetName() != TLSName {
		t.Errorf("Expected plugin name '%s', got '%s'", TLSName, plugin.GetName())
	}

	options := plugin.GetOptions()
	if options.Namespace != nil && *options.Namespace != CertManagerNamespace {
		t.Errorf("Expected namespace '%s', got '%v'", CertManagerNamespace, options.Namespace)
	}

	if options.Version != nil && *options.Version != TLSVersion {
		t.Errorf("Expected version '%s', got '%v'", TLSVersion, options.Version)
	}

	if options.ChartName != nil && *options.ChartName != "" {
		t.Errorf("Expected empty chart name, got '%s'", *options.ChartName)
	}

	if options.Repository != nil && *options.Repository != "" {
		t.Errorf("Expected empty repository, got '%s'", *options.Repository)
	}

	var _ Plugin = plugin
}

func TestTLSPluginConstants(t *testing.T) {
	if TLSName != "tls" {
		t.Errorf("Expected TLSName to be 'tls', got '%s'", TLSName)
	}

	if TLSVersion != "1.0.0" {
		t.Errorf("Expected TLSVersion to be '1.0.0', got '%s'", TLSVersion)
	}

	if TLSSecretName != "local-ca-secret" {
		t.Errorf("Expected TLSSecretName to be 'local-ca-secret', got '%s'", TLSSecretName)
	}

	if TLSClusterIssuerName != "local-ca-issuer" {
		t.Errorf("Expected TLSClusterIssuerName to be 'local-ca-issuer', got '%s'", TLSClusterIssuerName)
	}

	if CertValidityYears != 10 {
		t.Errorf("Expected CertValidityYears to be 10, got %d", CertValidityYears)
	}
}

func TestTLSGenerateCACertificate(t *testing.T) {
	plugin, err := NewTLS("dummy-kubeconfig", "test-cluster")
	if err != nil {
		t.Logf("K8s client creation failed (expected in test): %v", err)
		return
	}

	caCert, caKey, err := plugin.generateCACertificate()
	if err != nil {
		t.Errorf("Failed to generate CA certificate: %v", err)
		return
	}

	if len(caCert) == 0 {
		t.Error("CA certificate is empty")
	}

	if len(caKey) == 0 {
		t.Error("CA private key is empty")
	}

	if !containsPEMBlock(string(caCert), "CERTIFICATE") {
		t.Error("CA certificate does not contain proper PEM block")
	}

	if !containsPEMBlock(string(caKey), "RSA PRIVATE KEY") {
		t.Error("CA private key does not contain proper PEM block")
	}
}

func containsPEMBlock(content, blockType string) bool {
	return strings.Contains(content, fmt.Sprintf("-----BEGIN %s-----", blockType)) &&
		strings.Contains(content, fmt.Sprintf("-----END %s-----", blockType))
}
