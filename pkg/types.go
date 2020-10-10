package pkg

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

type PerfParams struct {
	KubeCfgPath  string
	ClientConfig clientcmd.ClientConfig
	ClientSet    *kubernetes.Clientset
}

func (params *PerfParams) Initialize() error {
	if params.ClientSet == nil {
		restConfig, err := params.RestConfig()
		if err != nil {
			return err
		}

		params.ClientSet, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			fmt.Println("failed to create client:", err)
			os.Exit(1)
		}
	}
	return nil
}

// RestConfig returns REST config, which can be to use to create specific clientset
func (params *PerfParams) RestConfig() (*rest.Config, error) {
	var err error

	if params.ClientConfig == nil {
		params.ClientConfig, err = params.GetClientConfig()
		if err != nil {
			return nil, err
		}
	}

	config, err := params.ClientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetClientConfig gets ClientConfig from KubeCfgPath
func (params *PerfParams) GetClientConfig() (clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if len(params.KubeCfgPath) == 0 {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}), nil
	}

	_, err := os.Stat(params.KubeCfgPath)
	if err == nil {
		loadingRules.ExplicitPath = params.KubeCfgPath
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}), nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	paths := filepath.SplitList(params.KubeCfgPath)
	if len(paths) > 1 {
		return nil, fmt.Errorf("Can not find config file. '%s' looks like a path. "+
			"Please use the env var KUBECONFIG if you want to check for multiple configuration files", params.KubeCfgPath)
	}
	return nil, fmt.Errorf("Config file '%s' can not be found", params.KubeCfgPath)
}
