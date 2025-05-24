package installer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

var settings = cli.New()

func NewHelm() (*HelmInstaller, error) {
	return &HelmInstaller{}, nil
}

type (
	HelmInstaller struct {
		ReleaseName  string
		ChartName    string
		RepoUrl      string
		RepoName     string
		Namespace    string
		ChartVersion string
		Values       map[string]interface{}
		KubeConfig   string
	}
)

func (h *HelmInstaller) Install(options *InstallOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	actionConfig, err := h.createHelmActionConfig(h.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create helm action config: %w", err)
	}

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(h.ReleaseName)

	if err == nil {

		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = h.Namespace

		chart, err := h.downloadAndLoadChart()
		if err != nil {
			return fmt.Errorf("failed to download and load chart: %w", err)
		}

		rel, err := upgrade.RunWithContext(ctx, h.ReleaseName, chart, h.Values)
		if err != nil {
			log.Printf("Error upgrading chart: %v\n", err)
			return fmt.Errorf("failed to upgrade chart: %w", err)
		}

		if rel == nil {
			return fmt.Errorf("failed to get release information")
		}

	} else {

		install := action.NewInstall(actionConfig)
		install.Namespace = h.Namespace
		install.ReleaseName = h.ReleaseName
		install.CreateNamespace = true

		chart, err := h.downloadAndLoadChart()
		if err != nil {
			return fmt.Errorf("failed to download and load chart: %w", err)
		}

		rel, err := install.RunWithContext(ctx, chart, h.Values)
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
	actionConfig, err := h.createHelmActionConfig(h.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create helm action config: %w", err)
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Timeout = 5 * time.Minute
	uninstall.Wait = true

	_, err = uninstall.Run(h.ReleaseName)
	if err != nil {
		log.Printf("Error uninstalling chart: %v\n", err)
		return fmt.Errorf("failed to uninstall chart: %w", err)
	}

	return nil
}

func (h *HelmInstaller) createHelmActionConfig(namespace string) (*action.Configuration, error) {
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("kubeconfig-%d", time.Now().UnixNano()))
	
	// Use more restrictive permissions for kubeconfig file
	if err := os.WriteFile(tmpPath, []byte(h.KubeConfig), 0600); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig to temp file: %w", err)
	}
	
	// Return the kubeconfig file path for explicit cleanup by the caller
	settings.KubeConfig = tmpPath
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	return actionConfig, nil
}

func (h *HelmInstaller) downloadAndLoadChart() (*chart.Chart, error) {
	chartPathOptions := action.ChartPathOptions{
		RepoURL:            h.RepoUrl,
		Version:            h.ChartVersion,
		PassCredentialsAll: true,
	}

	if err := h.addHelmRepo(); err != nil {
		return nil, fmt.Errorf("failed to add helm repository: %w", err)
	}

	chartPath, err := chartPathOptions.LocateChart(h.ChartName, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %s: %w", h.ChartName, err)
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

func (h *HelmInstaller) addHelmRepo() error {
	entry := repo.Entry{
		Name: h.RepoName,
		URL:  h.RepoUrl,
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
