package installer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mrgb7/playground/internal/k8s"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

var settings = cli.New()

func NewHelmInstaller(kubeConfig string) (*HelmInstaller, error) {
	return &HelmInstaller{
		KubeConfig: kubeConfig,
	}, nil
}

type HelmInstaller struct {
	KubeConfig string
}

func (h *HelmInstaller) Install(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	actionConfig, err := h.createHelmActionConfig(options.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create helm action config: %w", err)
	}

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(options.ApplicationName)

	if err == nil {
		// Release exists, upgrade it
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = options.Namespace

		chart, err := h.downloadAndLoadChart(options)
		if err != nil {
			return fmt.Errorf("failed to download and load chart: %w", err)
		}

		rel, err := upgrade.RunWithContext(ctx, options.ApplicationName, chart, options.Values)
		if err != nil {
			log.Printf("Error upgrading chart: %v\n", err)
			return fmt.Errorf("failed to upgrade chart: %w", err)
		}

		if rel == nil {
			return fmt.Errorf("failed to get release information")
		}
	} else {
		// Release doesn't exist, install it
		install := action.NewInstall(actionConfig)
		install.Namespace = options.Namespace
		install.ReleaseName = options.ApplicationName
		install.CreateNamespace = true

		chart, err := h.downloadAndLoadChart(options)
		if err != nil {
			return fmt.Errorf("failed to download and load chart: %w", err)
		}

		rel, err := install.RunWithContext(ctx, chart, options.Values)
		if err != nil {
			log.Printf("Error installing chart: %v\n", err)
			return fmt.Errorf("failed to install chart: %w", err)
		}

		if rel == nil {
			return fmt.Errorf("failed to get release information")
		}
	}
	return nil
}

func (h *HelmInstaller) UnInstall(options *InstallOptions) error {
	if options == nil {
		return fmt.Errorf("install options cannot be nil")
	}

	actionConfig, err := h.createHelmActionConfig(options.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create helm action config: %w", err)
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Timeout = 5 * time.Minute
	uninstall.Wait = true

	_, err = uninstall.Run(options.ApplicationName)
	if err != nil {
		log.Printf("Error uninstalling chart: %v\n", err)
		return fmt.Errorf("failed to uninstall chart: %w", err)
	}

	k8sClient, err := k8s.NewK8sClient(h.KubeConfig)
	if err != nil {
		log.Printf("Warning: Failed to create k8s client: %v\n", err)
	} else if err := k8sClient.DeleteNamespace(options.Namespace); err != nil {
		log.Printf("Warning: Failed to cleanup namespace: %v\n", err)
	}

	return nil
}

func (h *HelmInstaller) createHelmActionConfig(namespace string) (*action.Configuration, error) {
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("kubeconfig-%d", time.Now().UnixNano()))

	if err := os.WriteFile(tmpPath, []byte(h.KubeConfig), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig to temp file: %w", err)
	}

	settings.KubeConfig = tmpPath
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	return actionConfig, nil
}

func (h *HelmInstaller) downloadAndLoadChart(options *InstallOptions) (*chart.Chart, error) {
	chartPathOptions := action.ChartPathOptions{
		RepoURL:            options.RepoURL,
		Version:            options.Version,
		PassCredentialsAll: true,
	}

	if err := h.addHelmRepo(options); err != nil {
		return nil, fmt.Errorf("failed to add helm repository: %w", err)
	}

	chartPath, err := chartPathOptions.LocateChart(*options.ChartName, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %s: %w", *options.ChartName, err)
	}

	log.Printf("Chart found at: %s\n", chartPath)

	loadedChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	if loadedChart.Metadata.Name == "" {
		return nil, fmt.Errorf("chart has no name")
	}

	if loadedChart.Metadata.Version == "" {
		return nil, fmt.Errorf("chart has no version")
	}

	log.Printf("Successfully loaded chart: %s version %s\n", loadedChart.Metadata.Name, loadedChart.Metadata.Version)

	return loadedChart, nil
}

func (h *HelmInstaller) addHelmRepo(options *InstallOptions) error {
	entry := repo.Entry{
		Name: options.RepoName,
		URL:  options.RepoURL,
	}

	r, err := repo.NewChartRepository(&entry, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	indexFilePath, err := r.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("failed to download repository index: %w", err)
	}

	_, err = repo.LoadIndexFile(indexFilePath)
	if err != nil {
		return fmt.Errorf("failed to load repository index: %w", err)
	}

	return nil
}
