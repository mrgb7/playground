package plugin

import (
	"context"

	"github.com/mrgb7/playground/internal/plugins"
	"github.com/mrgb7/playground/pkg/logger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove plugin",
	Long:  `Remove plugin from the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		uninstallOperation := func(plugin plugins.Plugin, kubeConfig, clusterName string) error {
			if err := plugin.Uninstall(kubeConfig, clusterName); err != nil {
				return err
			}

			namespace := pName
			if err := deleteNamespace(kubeConfig, namespace); err != nil {
				logger.Warnf("Failed to delete namespace %s: %v", namespace, err)
			}
			return nil
		}

		_ = executePluginOperation(pName, cName, uninstallOperation,
			"Successfully uninstalled %s", "Error uninstalling plugin")
	},
}

func deleteNamespace(kubeConfigData, namespace string) error {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfigData))
	if err != nil {
		logger.Errorf("Failed to parse kubeconfig data: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("Failed to create kubernetes client: %v", err)

	}

	return clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
}

func init() {
	flags := removeCmd.Flags()
	flags.StringVarP(&pName, "name", "n", "", "Name of the plugin")
	flags.StringVarP(&cName, "cluster", "c", "", "Name of the cluster")
	if err := removeCmd.MarkFlagRequired("name"); err != nil {
		logger.Errorln("Failed to mark name flag as required: %v", err)
	}
	if err := removeCmd.MarkFlagRequired("cluster"); err != nil {
		logger.Errorln("Failed to mark cluster flag as required: %v", err)
	}
	PluginCmd.AddCommand(removeCmd)
}
