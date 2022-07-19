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

func TestBootstrapConfig(t *testing.T) {
	configYaml := `
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
    wait:
    timeout:
    template:
`

	configFile, cleanup := setupConfig(t, configYaml)
	defer cleanup()

	err := BootstrapConfig()
	assert.NilError(t, err)
	assert.Equal(t, globalConfig.ConfigFile(), configFile)
}

func TestBootstrapConfigWithoutConfigFile(t *testing.T) {
	_, cleanup := setupConfig(t, "")
	defer cleanup()

	err := BootstrapConfig()
	assert.NilError(t, err)
	assert.Equal(t, globalConfig.ConfigFile(), bootstrapDefaults.configFile)
}

func setupConfig(t *testing.T, configContent string) (string, func()) {
	// Avoid being fooled by the things in the real homedir
	tmpHome := "/tmp"
	tmpPath := "/tmp/.config/kperf/"
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	// Save old args
	backupArgs := os.Args

	// Write out a temporary configContent file
	var cfgFile string
	if configContent != "" {
		cfgFile = filepath.Join(tmpPath, "config.yaml")
		err := os.MkdirAll(tmpPath, os.ModePerm)
		if err != nil {
			fmt.Println("failed to mkdir")
		}
		os.Args = []string{"kperf", "--config", cfgFile}
		err = ioutil.WriteFile(cfgFile, []byte(configContent), 0644)
		assert.NilError(t, err)
	}

	// Reset various global state
	oldHomeDirDisableCache := homedir.DisableCache
	homedir.DisableCache = true
	viper.Reset()
	globalConfig = config{}
	bootstrapDefaults = initDefaults()
	return cfgFile, func() {
		// Cleanup everything
		os.Setenv("HOME", oldHome)
		os.Args = backupArgs
		bootstrapDefaults = initDefaults()
		viper.Reset()
		homedir.DisableCache = oldHomeDirDisableCache
		globalConfig = config{}
	}
}
