package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	Clientset *kubernetes.Clientset
	Dynamic   *dynamic.DynamicClient
	Config    *rest.Config
}

func NewK8sClient(kubeConfig string) (*K8sClient, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &K8sClient{
		Clientset: clientset,
		Dynamic:   dynamicClient,
		Config:    restConfig,
	}, nil
}

func (k *K8sClient) GetNameSpace(name string, ctx context.Context) (string, error) {
	namespace, err := k.Clientset.CoreV1().Namespaces().Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return "", err
	}
	return namespace.Name, nil
}

func (k *K8sClient) DeleteNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}

	ns, err := k.Clientset.CoreV1().Namespaces().Get(context.Background(), namespace, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("error checking namespace: %w", err)
	}

	if ns.Status.Phase == corev1.NamespaceTerminating {
		return k.waitForNamespaceDeletion(namespace)
	}

	err = k.Clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, v1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting namespace: %w", err)
	}

	return k.waitForNamespaceDeletion(namespace)
}

func (c *K8sClient) waitForNamespaceDeletion(namespace string) error {
	timeout := 5 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for namespace deletion after %v", timeout)
		case <-ticker.C:
			_, err := c.Clientset.CoreV1().Namespaces().Get(context.Background(), namespace, v1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("error checking namespace status: %w", err)
			}
		}
	}
}
