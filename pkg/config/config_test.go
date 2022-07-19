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

func setupConfig(t *testing.T, configContent string) (string, func()) {
	tmpPath := "./config.yaml"
	globalConfig.configFile = tmpPath

	// Save old args
	backupArgs := os.Args

	// Write out a temporary configContent file
	if configContent != "" {
		os.Args = []string{"kperf", "--config", tmpPath}
		err := ioutil.WriteFile(tmpPath, []byte(configContent), 0644)
		if err != nil {
			fmt.Println(err)
			return "", nil
		}
	}

	// Reset various global state
	oldHomeDirDisableCache := homedir.DisableCache
	homedir.DisableCache = true
	viper.Reset()
	bootstrapDefaults = initDefaults()
	return tmpPath, func() {
		// Cleanup everything
		os.Args = backupArgs
		bootstrapDefaults = initDefaults()
		viper.Reset()
		homedir.DisableCache = oldHomeDirDisableCache
		globalConfig = config{}
	}
}
