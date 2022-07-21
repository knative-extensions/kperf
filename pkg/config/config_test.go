// Copyright © 2022 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"gotest.tools/v3/assert"
)

const (
	FakeConfigYaml = `
service:
  generate:
    number:
    interval:
    batch:
    concurrency:
    min-scale:
    max-scale:
    namespace:
    namespace-prefix:
    namespace-range:
    svc-prefix:
`
)

func TestBootstrapConfig(t *testing.T) {
	t.Run("config is not set and default config file doesn't exist", func(t *testing.T) {
		_, cleanup := setupConfig(t, "", "")
		defer cleanup()

		err := BootstrapConfig()
		assert.NilError(t, err)
		assert.Equal(t, globalConfig.ConfigFile(), defaultConfigLocation("config.yaml"))
	})
	t.Run("config is not set and default config file exists", func(t *testing.T) {
		_, cleanup := setupConfig(t, FakeConfigYaml, "")
		defer cleanup()

		err := BootstrapConfig()
		assert.NilError(t, err)
		assert.Equal(t, globalConfig.ConfigFile(), defaultConfigLocation("config.yaml"))
	})
	t.Run("config is set", func(t *testing.T) {
		fakePath := "./abc/config.yaml"
		configFile, cleanup := setupConfig(t, FakeConfigYaml, fakePath)
		defer cleanup()

		err := BootstrapConfig()
		assert.NilError(t, err)
		assert.Equal(t, globalConfig.ConfigFile(), configFile)
	})
}

func setupConfig(t *testing.T, configContent string, configPath string) (string, func()) {
	// Avoid to be fooled by the things in real homedir
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	homedir.DisableCache = true
	// Save old args
	backupArgs := os.Args

	// Write out a temporary configContent file
	var cfgFile string
	if configPath != "" {
		cfgFile = filepath.Join(tmpDir, configPath)
		if configContent != "" {
			err := os.MkdirAll(filepath.Dir(cfgFile), 0775)
			assert.NilError(t, err)

			err = ioutil.WriteFile(cfgFile, []byte(configContent), 0644)
			assert.NilError(t, err)

			os.Args = []string{"kperf", "--config", cfgFile}
			globalConfig = config{cfgFile}
		}
	} else {
		cfgFile = defaultConfigLocation("config.yaml")
		if configContent != "" {
			err := os.MkdirAll(filepath.Dir(cfgFile), 0775)
			assert.NilError(t, err)

			err = ioutil.WriteFile(cfgFile, []byte(configContent), 0644)
			assert.NilError(t, err)

			os.Args = []string{"kperf"}
			globalConfig = config{cfgFile}
		} else {
			os.Args = []string{"kperf"}
			globalConfig = config{cfgFile}
		}
	}

	// Reset various global state
	oldHomeDirDisableCache := homedir.DisableCache
	viper.Reset()
	bootstrapDefaults = initDefaults()

	return cfgFile, func() {
		// Cleanup everything
		os.Setenv("HOME", oldHome)
		os.Args = backupArgs
		bootstrapDefaults = initDefaults()
		viper.Reset()
		homedir.DisableCache = oldHomeDirDisableCache
		globalConfig = config{}
		if configContent != "" {
			if err := os.RemoveAll(tmpDir); err != nil {
				fmt.Println("failed to clean up temp file or directory:", err)
			}
		}
	}
}
