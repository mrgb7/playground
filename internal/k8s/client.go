package k8s

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	Clientset *kubernetes.Clientset
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

	return &K8sClient{
		Clientset: clientset,
	}, nil
}

func (k *K8sClient) GetNameSpace(name string, ctx context.Context) (string, error) {
	namespace, err := k.Clientset.CoreV1().Namespaces().Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return "", err
	}
	return namespace.Name, nil
}
