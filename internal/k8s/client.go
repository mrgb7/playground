package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/mrgb7/playground/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	Clientset              *kubernetes.Clientset
	Dynamic                *dynamic.DynamicClient
	apiextensionsclientset *apiextensionsclientset.Clientset
	Config                 *rest.Config
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
	apiextensionsClient, err := apiextensionsclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	return &K8sClient{
		Clientset:              clientset,
		Dynamic:                dynamicClient,
		apiextensionsclientset: apiextensionsClient,
		Config:                 restConfig,
	}, nil
}

func (k *K8sClient) GetNameSpace(name string, ctx context.Context) (string, error) {
	namespace, err := k.Clientset.CoreV1().
		Namespaces().Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return "", err
	}
	return namespace.Name, nil
}

func (k *K8sClient) DeleteNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}

	ns, err := k.Clientset.CoreV1().
		Namespaces().
		Get(context.Background(), namespace, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("error checking namespace: %w", err)
	}

	if ns.Status.Phase == corev1.NamespaceTerminating {
		return k.waitForNamespaceDeletion(namespace)
	}

	err = k.Clientset.CoreV1().
		Namespaces().
		Delete(context.Background(), namespace, v1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting namespace: %w", err)
	}

	return k.waitForNamespaceDeletion(namespace)
}

func (k *K8sClient) GetCRDsByGroup(group string) ([]string, error) {
	if k.apiextensionsclientset == nil {
		return nil, fmt.Errorf("apiextensions client is not initialized")
	}

	crdList, err := k.apiextensionsclientset.ApiextensionsV1().
		CustomResourceDefinitions().
		List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRDs for group %s: %w", group, err)
	}

	if len(crdList.Items) == 0 {
		return nil, fmt.Errorf("no CRDs found for group %s", group)
	}

	var crds []string
	for _, item := range crdList.Items {
		if item.Spec.Group == group {
			crds = append(crds, item.Name)
		}
	}
	return crds, nil
}

func (k *K8sClient) DeleteCRDsGroup(group string) error {
	crds, err := k.GetCRDsByGroup(group)
	if err != nil {
		return fmt.Errorf("failed to get CRDs for group %s: %w", group, err)
	}

	for _, crd := range crds {
		err = k.apiextensionsclientset.
			ApiextensionsV1().
			CustomResourceDefinitions().
			Delete(context.Background(), crd, v1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete CRD %s: %w", crd, err)
		}
	}

	return nil
}

func (k *K8sClient) EnsureApp(namespace, appName string) <-chan error {
	logger.Infof("Ensuring app %s in namespace %s", appName, namespace)
	doneCh := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		timer := time.NewTimer(5 * time.Minute)
		for {
			select {
			case <-ticker.C:
				deploys, err := k.Clientset.AppsV1().
					Deployments(namespace).
					List(context.Background(),
						v1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", appName)})
				if err != nil {
					continue
				}
				if len(deploys.Items) == 0 {
					continue
				}
				allReady := true
				for _, deploy := range deploys.Items {
					if deploy.Status.ReadyReplicas < deploy.Status.Replicas || deploy.Status.Replicas <= 0 {
						logger.Debugf("Deployment %s in namespace %s is not ready yet", deploy.Name, namespace)
						allReady = false
						break
					}
				}
				if !allReady {
					logger.Debugf("App %s in namespace %s is not ready yet", appName, namespace)
					continue
				}
				doneCh <- nil

			case <-timer.C:
				doneCh <- fmt.Errorf("timeout waiting for app %s in namespace %s to be ready", appName, namespace)
				return
			}
		}
	}()

	return doneCh
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
			_, err := c.Clientset.CoreV1().
				Namespaces().
				Get(context.Background(), namespace, v1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("error checking namespace status: %w", err)
			}
		}
	}
}
