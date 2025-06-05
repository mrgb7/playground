package plugins

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"github.com/mrgb7/playground/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	TLSName              = "tls"
	TLSVersion           = "1.0.0"
	TLSSecretName        = "local-ca-secret"
	TLSClusterIssuerName = "local-ca-issuer"
	CertValidityYears    = 10
	RSAKeySize           = 4096
)

type TLS struct {
	KubeConfig  string
	k8sClient   *k8s.K8sClient
	ClusterName string
	*BasePlugin
}

func NewTLS(kubeConfig, clusterName string) (*TLS, error) {
	c, err := k8s.NewK8sClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	tls := &TLS{
		KubeConfig:  kubeConfig,
		k8sClient:   c,
		ClusterName: clusterName,
	}
	tls.BasePlugin = NewBasePlugin(kubeConfig, tls)
	return tls, nil
}

func (t *TLS) GetName() string {
	return TLSName
}

func (t *TLS) GetOptions() PluginOptions {
	return PluginOptions{
		Version:     &TLSVersion,
		Namespace:   &CertManagerNamespace,
		ChartName:   &TLSName,
		RepoName:    &CertManagerRepoName,
		Repository:  &CertManagerRepoURL,
		releaseName: &TLSName,
	}
}

func (t *TLS) Install(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Infoln("Installing TLS plugin for cluster: %s", clusterName)

	caCert, caKey, err := t.generateCACertificate()
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %w", err)
	}

	// Validate the generated certificate
	if err := t.validateCACertificate(caCert); err != nil {
		logger.Warnln("Certificate validation warning: %v", err)
	}

	if err := t.storeCASecret(caCert, caKey); err != nil {
		return fmt.Errorf("failed to store CA secret: %w", err)
	}

	if err := t.createClusterIssuer(); err != nil {
		return fmt.Errorf("failed to create cluster issuer: %w", err)
	}

	if err := t.printTrustInstructions(caCert); err != nil {
		return fmt.Errorf("failed to print trust instructions: %w", err)
	}

	logger.Successln("TLS plugin installed successfully")
	return nil
}

func (t *TLS) Uninstall(kubeConfig, clusterName string, ensure ...bool) error {
	logger.Infoln("Uninstalling TLS plugin")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := t.k8sClient.Clientset.CoreV1().Secrets(CertManagerNamespace).Delete(
		ctx, TLSSecretName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		logger.Warnln("Failed to delete CA secret: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "clusterissuers",
	}
	err = t.k8sClient.Dynamic.Resource(gvr).Delete(
		ctx, TLSClusterIssuerName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		logger.Warnln("Failed to delete cluster issuer: %v", err)
	}

	logger.Successln("TLS plugin uninstalled successfully")
	return nil
}

func (t *TLS) Status() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := t.k8sClient.Clientset.CoreV1().Secrets(CertManagerNamespace).Get(
		ctx, TLSSecretName, metav1.GetOptions{})
	if err != nil {
		return "TLS CA secret not found"
	}

	gvr := schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "clusterissuers",
	}
	_, err = t.k8sClient.Dynamic.Resource(gvr).Get(
		ctx, TLSClusterIssuerName, metav1.GetOptions{})
	if err != nil {
		return "TLS cluster issuer not found"
	}

	return "TLS is configured and ready"
}

func (t *TLS) generateCACertificate() ([]byte, []byte, error) {
	logger.Infoln("Generating CA certificate for domain: *.%s.local", t.ClusterName)

	privateKey, err := rsa.GenerateKey(rand.Reader, RSAKeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{fmt.Sprintf("%s Local CA", t.ClusterName)},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    fmt.Sprintf("%s Local CA", t.ClusterName),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(CertValidityYears, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		DNSNames: []string{
			fmt.Sprintf("*.%s.local", t.ClusterName),
			fmt.Sprintf("%s.local", t.ClusterName),
			"localhost",
		},
		IPAddresses: []net.IP{
			net.IPv4(127, 0, 0, 1),
			net.IPv6loopback,
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	logger.Successln("CA certificate generated successfully")
	return certPEM, keyPEM, nil
}

func (t *TLS) storeCASecret(caCert, caKey []byte) error {
	logger.Infoln("Storing CA certificate in Kubernetes secret")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TLSSecretName,
			Namespace: CertManagerNamespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": caCert,
			"tls.key": caKey,
		},
	}

	_, err := t.k8sClient.Clientset.CoreV1().Secrets(CertManagerNamespace).Create(ctx, secret, metav1.CreateOptions{})
	switch {
	case err != nil && strings.Contains(err.Error(), "already exists"):
		// Get the existing secret to preserve metadata
		existing, getErr := t.k8sClient.Clientset.CoreV1().Secrets(CertManagerNamespace).Get(ctx, TLSSecretName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing CA secret: %w", getErr)
		}

		// Preserve the existing metadata and update only the data
		secret.ResourceVersion = existing.ResourceVersion
		secret.UID = existing.UID
		secret.CreationTimestamp = existing.CreationTimestamp
		secret.Generation = existing.Generation

		// Copy any existing labels and annotations
		if existing.Labels != nil {
			secret.Labels = existing.Labels
		}
		if existing.Annotations != nil {
			secret.Annotations = existing.Annotations
		}

		_, err = t.k8sClient.Clientset.CoreV1().Secrets(CertManagerNamespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update existing CA secret: %w", err)
		}
		logger.Infoln("Updated existing CA secret")
	case err != nil:
		return fmt.Errorf("failed to create CA secret: %w", err)
	default:
		logger.Successln("Created CA secret successfully")
	}

	return nil
}

func (t *TLS) createClusterIssuer() error {
	logger.Infoln("Creating cluster issuer: %s", TLSClusterIssuerName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gvr := schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "clusterissuers",
	}

	clusterIssuer := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "ClusterIssuer",
			"metadata": map[string]interface{}{
				"name": TLSClusterIssuerName,
			},
			"spec": map[string]interface{}{
				"ca": map[string]interface{}{
					"secretName": TLSSecretName,
				},
			},
		},
	}

	_, err := t.k8sClient.Dynamic.Resource(gvr).Create(ctx, clusterIssuer, metav1.CreateOptions{})
	switch {
	case err != nil && strings.Contains(err.Error(), "already exists"):
		// Get the existing cluster issuer to preserve metadata
		existing, getErr := t.k8sClient.Dynamic.Resource(gvr).Get(ctx, TLSClusterIssuerName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing cluster issuer: %w", getErr)
		}

		// Preserve the existing metadata and update only the spec
		clusterIssuer.SetResourceVersion(existing.GetResourceVersion())
		clusterIssuer.SetUID(existing.GetUID())
		clusterIssuer.SetCreationTimestamp(existing.GetCreationTimestamp())
		clusterIssuer.SetGeneration(existing.GetGeneration())

		// Copy any existing labels and annotations
		if labels := existing.GetLabels(); labels != nil {
			clusterIssuer.SetLabels(labels)
		}
		if annotations := existing.GetAnnotations(); annotations != nil {
			clusterIssuer.SetAnnotations(annotations)
		}

		_, err = t.k8sClient.Dynamic.Resource(gvr).Update(ctx, clusterIssuer, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update existing cluster issuer: %w", err)
		}
		logger.Infoln("Updated existing cluster issuer")
	case err != nil:
		return fmt.Errorf("failed to create cluster issuer: %w", err)
	default:
		logger.Successln("Created cluster issuer successfully")
	}

	return nil
}

func (t *TLS) printMacOSInstructions(tempFile *os.File) {
	logger.Infoln("🍎 macOS Trust Instructions:")
	logger.Infoln("")
	logger.Infoln("Method 1 - Command Line (Recommended):")
	logger.Infoln("sudo security add-trusted-cert -d -r trustRoot \\")
	logger.Infoln("  -k /Library/Keychains/System.keychain %s", tempFile.Name())
	logger.Infoln("")
	logger.Infoln("Method 2 - GUI Method:")
	logger.Infoln("1. Double-click the certificate file to open Keychain Access")
	logger.Infoln("2. Select 'System' keychain (important for system-wide trust)")
	logger.Infoln("3. Find the certificate and double-click it")
	logger.Infoln("4. Expand 'Trust' section")
	logger.Infoln("5. Set 'When using this certificate' to 'Always Trust'")
	logger.Infoln("6. Set 'Secure Sockets Layer (SSL)' to 'Always Trust'")
	logger.Infoln("7. Close the dialog and enter your admin password")
	logger.Infoln("")
	logger.Infoln("Method 3 - For Chrome Compatibility:")
	logger.Infoln("1. Open Chrome and go to chrome://settings/security")
	logger.Infoln("2. Click 'Manage certificates'")
	logger.Infoln("3. Go to 'Authorities' tab")
	logger.Infoln("4. Click 'Import' and select the certificate file")
	logger.Infoln("5. Check 'Trust this certificate for identifying websites'")
	logger.Infoln("6. Click 'OK'")
	logger.Infoln("")
	logger.Infoln("⚠️  Important Notes:")
	logger.Infoln("- After trusting the certificate, restart Chrome completely")
	logger.Infoln("- Clear Chrome's cache (chrome://settings/clearBrowserData)")
	logger.Infoln("- Make sure you're accessing sites with the exact domain: *.%s.local", t.ClusterName)
	logger.Infoln("- For localhost testing, use: https://localhost or https://127.0.0.1")
}

func (t *TLS) printLinuxInstructions(tempFile *os.File) {
	logger.Infoln("🐧 Linux Trust Instructions:")
	logger.Infoln("sudo cp %s /usr/local/share/ca-certificates/%s-ca.crt", tempFile.Name(), t.ClusterName)
	logger.Infoln("sudo update-ca-certificates")
	logger.Infoln("")
	logger.Infoln("For Firefox (if needed):")
	logger.Infoln("Import the certificate manually in Firefox preferences > Privacy & Security > Certificates")
}

func (t *TLS) printWindowsInstructions(tempFile *os.File) {
	logger.Infoln("🪟 Windows Trust Instructions:")
	logger.Infoln("certlm.msc")
	logger.Infoln("1. Right-click 'Trusted Root Certification Authorities'")
	logger.Infoln("2. Select 'All Tasks' > 'Import'")
	logger.Infoln("3. Browse and select: %s", tempFile.Name())
	logger.Infoln("4. Place in 'Trusted Root Certification Authorities' store")
	logger.Infoln("")
	logger.Infoln("Alternative (PowerShell as Administrator):")
	logger.Infoln("Import-Certificate -FilePath \"%s\" -CertStoreLocation Cert:\\LocalMachine\\Root", tempFile.Name())
}

func (t *TLS) printGenericInstructions(tempFile *os.File) {
	logger.Infoln("📋 Generic Trust Instructions:")
	logger.Infoln("Add the following certificate to your system's trusted CA store:")
	logger.Infoln("Certificate file: %s", tempFile.Name())
}

func (t *TLS) printTrustInstructions(caCert []byte) error {
	logger.Infoln("Generating trust instructions for your operating system")

	tempFile, err := os.CreateTemp("", fmt.Sprintf("%s-ca-*.crt", t.ClusterName))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := tempFile.Close(); err != nil {
			logger.Debugln("Failed to close temporary file: %v", err)
		}
	}()

	if _, err := tempFile.Write(caCert); err != nil {
		return fmt.Errorf("failed to write certificate to temp file: %w", err)
	}

	logger.Infoln("")
	logger.Infoln("🔐 CA Certificate has been generated!")
	logger.Infoln("📍 Temporary certificate file: %s", tempFile.Name())
	logger.Infoln("")

	switch runtime.GOOS {
	case "darwin":
		t.printMacOSInstructions(tempFile)
	case "linux":
		t.printLinuxInstructions(tempFile)
	case "windows":
		t.printWindowsInstructions(tempFile)
	default:
		t.printGenericInstructions(tempFile)
	}

	logger.Infoln("")
	logger.Infoln("🎯 Certificate Details:")
	logger.Infoln("Domain: *.%s.local", t.ClusterName)
	logger.Infoln("Validity: %d years", CertValidityYears)
	logger.Infoln("Cluster Issuer: %s", TLSClusterIssuerName)
	logger.Infoln("")
	logger.Infoln("🚀 You can now use TLS certificates in your cluster!")
	logger.Infoln("Example ingress annotation: cert-manager.io/cluster-issuer: %s", TLSClusterIssuerName)

	logger.Infoln("")
	logger.Infoln("🔧 Troubleshooting Chrome Issues:")
	logger.Infoln("If Chrome still shows certificate warnings after trusting the CA:")
	logger.Infoln("1. Ensure you've restarted Chrome completely (quit all instances)")
	logger.Infoln("2. Clear Chrome's SSL cache: chrome://settings/clearBrowserData")
	logger.Infoln("3. Check certificate is in Chrome: chrome://settings/certificates")
	logger.Infoln("4. Verify domain matches exactly: https://%s.local or https://subdomain.%s.local", t.ClusterName, t.ClusterName)
	logger.Infoln("5. Try incognito mode to test without cache")
	logger.Infoln("6. Check Chrome's certificate viewer: Developer Tools > Security tab")
	logger.Infoln("7. For local development, ensure your app serves HTTPS on the correct domain")

	logger.Infoln("")
	logger.Infoln("📋 Certificate content (base64):")
	certBase64 := base64.StdEncoding.EncodeToString(caCert)
	logger.Infoln(certBase64)

	return nil
}

func (t *TLS) GetClusterIssuerName() string {
	return TLSClusterIssuerName
}

func (t *TLS) GetDependencies() []string {
	return []string{"cert-manager"} // TLS depends on cert-manager
}

func (t *TLS) validateCACertificate(certPEM []byte) error {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	logger.Infoln("🔍 Certificate Validation Results:")
	logger.Infoln("Subject: %s", cert.Subject.String())
	logger.Infoln("Issuer: %s", cert.Issuer.String())
	logger.Infoln("Serial Number: %s", cert.SerialNumber.String())
	logger.Infoln("Valid From: %s", cert.NotBefore.Format(time.RFC3339))
	logger.Infoln("Valid To: %s", cert.NotAfter.Format(time.RFC3339))
	logger.Infoln("Is CA: %t", cert.IsCA)

	logger.Infoln("Key Usage:")
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		logger.Infoln("  - Certificate Signing")
	}
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		logger.Infoln("  - Digital Signature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		logger.Infoln("  - Key Encipherment")
	}

	logger.Infoln("Extended Key Usage:")
	for _, eku := range cert.ExtKeyUsage {
		switch eku {
		case x509.ExtKeyUsageServerAuth:
			logger.Infoln("  - Server Authentication")
		case x509.ExtKeyUsageClientAuth:
			logger.Infoln("  - Client Authentication")
		default:
			logger.Infoln("  - Other: %v", eku)
		}
	}

	logger.Infoln("DNS Names:")
	for _, dns := range cert.DNSNames {
		logger.Infoln("  - %s", dns)
	}

	logger.Infoln("IP Addresses:")
	for _, ip := range cert.IPAddresses {
		logger.Infoln("  - %s", ip.String())
	}

	// Check for common Chrome compatibility issues
	if len(cert.DNSNames) == 0 {
		logger.Warnln("⚠️  WARNING: No DNS names in Subject Alternative Name - Chrome may reject certificates")
	}

	if !cert.IsCA {
		logger.Warnln("⚠️  WARNING: Certificate is not marked as CA")
	}

	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		logger.Warnln("⚠️  WARNING: Certificate cannot sign other certificates")
	}

	return nil
}
