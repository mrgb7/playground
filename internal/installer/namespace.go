package installer

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type NamespaceManager struct {
	kubeConfig string
}

func NewNamespaceManager(kubeConfig string) *NamespaceManager {
	return &NamespaceManager{
		kubeConfig: kubeConfig,
	}
}

func (nm *NamespaceManager) DeleteNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(nm.kubeConfig))
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	err = clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Namespace doesn't exist, which is fine
		}
		return fmt.Errorf("failed to delete namespace %s: %w", namespace, err)
	}

	for {
		_, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("error checking namespace status: %w", err)
		}
		time.Sleep(time.Second)
	}
}
