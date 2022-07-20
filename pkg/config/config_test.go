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
	t.Run("cannot stat configFile", func(t *testing.T) {
		backupArgs := os.Args
		os.Args = []string{"kperf", "--config", "\000x"}
		defer func() {
			os.Args = backupArgs
			viper.Reset()
		}()
		err := BootstrapConfig()
		assert.ErrorContains(t, err, "cannot stat configfile")
	})
	t.Run("configFile is none", func(t *testing.T) {
		_, cleanup := setupConfig(t, "")
		defer cleanup()

		err := BootstrapConfig()
		assert.NilError(t, err)
		assert.Equal(t, globalConfig.ConfigFile(), bootstrapDefaults.configFile)
	})
	t.Run("configFile is not none", func(t *testing.T) {
		configFile, cleanup := setupConfig(t, FakeConfigYaml)
		defer cleanup()

		err := BootstrapConfig()
		assert.NilError(t, err)
		assert.Equal(t, globalConfig.ConfigFile(), configFile)
	})
}

func setupConfig(t *testing.T, configContent string) (string, func()) {
	tmpDir := t.TempDir()

	// Avoid to be fooled by the things in real homedir
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Save old args
	backupArgs := os.Args

	// Write out a temporary configContent file
	var cfgFile string
	if configContent != "" {
		cfgFile = filepath.Join(tmpDir, "./.config/kperf/config.yaml")
		os.Args = []string{"kperf", "--config", cfgFile}

		err := os.MkdirAll(filepath.Dir(cfgFile), 0775)
		assert.NilError(t, err)

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
		if configContent != "" {
			if err := os.RemoveAll(tmpDir); err != nil {
				fmt.Println("failed to clean up temp file or directory:", err)
			}
		}
	}
}

//func setupConfig1(t *testing.T, configContent string) (string, func()) {
//	tmpPath := "./config.yaml"
//	globalConfig.configFile = tmpPath
//
//	// Save old args
//	backupArgs := os.Args
//
//	// Write out a temporary configContent file
//	if configContent != "" {
//		tmpPath := "./config.yaml"
//		globalConfig.configFile = tmpPath
//		os.Args = []string{"kperf", "--config", tmpPath}
//		err := ioutil.WriteFile(tmpPath, []byte(configContent), 0644)
//		if err != nil {
//			fmt.Println(err)
//			return "", nil
//		}
//	}
//
//	// Reset various global state
//	oldHomeDirDisableCache := homedir.DisableCache
//	homedir.DisableCache = true
//	viper.Reset()
//	bootstrapDefaults = initDefaults()
//	return tmpPath, func() {
//		// Cleanup everything
//		os.Args = backupArgs
//		bootstrapDefaults = initDefaults()
//		viper.Reset()
//		homedir.DisableCache = oldHomeDirDisableCache
//		globalConfig = config{}
//	}
//}
