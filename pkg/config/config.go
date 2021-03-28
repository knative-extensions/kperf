// Copyright 2021 The Knative Authors
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
	"github.com/mitchellh/go-homedir"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
)

type config struct {
	// configFile is the config file location
	configFile string
}

// bootstrapDefaults are the defaults values to use
type defaultConfig struct {
	configFile string
}

// Initialize defaults
var bootstrapDefaults = initDefaults()

// ConfigFile returns the config file which is either the default XDG conform
// config file location or the one set with --config
func (c *config) ConfigFile() string {
	if c.configFile != "" {
		return c.configFile
	}
	return bootstrapDefaults.configFile
}

// Config used for flag binding
var globalConfig = config{}

// GlobalConfig is the global configuration available for every sub-command
var GlobalConfig = &globalConfig

func BootstrapConfig() error {
	// Check if configfile exists. If not, just return
	configFile := GlobalConfig.ConfigFile()
	_, err := os.Lstat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file to read
			return nil
		}
		return fmt.Errorf("cannot stat configfile %s: %w", configFile, err)
	}

	viper.SetConfigFile(GlobalConfig.ConfigFile())
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	setDefault()

	return nil
}

// Add bootstrap flags use in a separate bootstrap proceeds
func AddBootstrapFlags(flags *flag.FlagSet) {
	flags.StringVar(&globalConfig.configFile, "config", "", fmt.Sprintf("kperf configuration file (default: %s)", defaultConfigFileForUsageMessage()))
}

// Prepare the default config file for the usage message
func defaultConfigFileForUsageMessage() string {
	if runtime.GOOS == "windows" {
		return "%APPDATA%\\kperf\\config.yaml"
	}
	return "~/.config/kperf/config.yaml"
}

// Initialize defaults. This happens lazily go allow to change the
// home directory for e.g. tests
func initDefaults() *defaultConfig {
	return &defaultConfig{
		configFile: defaultConfigLocation("config.yaml"),
	}
}

func defaultConfigLocation(subpath string) string {
	var base string
	if runtime.GOOS == "windows" {
		base = defaultConfigDirWindows()
	} else {
		base = defaultConfigDirUnix()
	}
	return filepath.Join(base, subpath)
}

func defaultConfigDirUnix() string {
	home, err := homedir.Dir()
	if err != nil {
		home = "~"
	}

	// Check if config file existed
	if configHome := filepath.Join(home, ".config", ".kperf"); dirExists(configHome) {
		return configHome
	}

	// Respect XDG_CONFIG_HOME if set
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		return filepath.Join(xdgHome, "kperf")
	}
	// Fallback to XDG default for both Linux and macOS
	// ~/.config/kperf
	return filepath.Join(home, ".config", "kperf")
}

func defaultConfigDirWindows() string {
	home, err := homedir.Dir()
	if err != nil {
		// Check if config file existed
		if configHome := filepath.Join(home, ".config", ".kperf"); dirExists(configHome) {
			return configHome
		}
	}

	return filepath.Join(os.Getenv("APPDATA"), "kperf")
}

func dirExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func setDefault() {
	viper.SetDefault("kperf.output", ".")

	viper.SetDefault("svc_provision.number", "100")
	viper.SetDefault("svc_provision.interval", "1")
	viper.SetDefault("svc_provision.batch", "1")

	viper.SetDefault("svc_provision.concurrency", "10")
	viper.SetDefault("svc_provision.min_scale", "0")
	viper.SetDefault("svc_provision.max_scale", "1")

	viper.SetDefault("svc_provision.namespace_prefix", "")
	viper.SetDefault("svc_provision.namespace_range", "")
	viper.SetDefault("svc_provision.namespace", "default")

	viper.SetDefault("svc_provision.svc_prefix", "kperfsvc")

	viper.SetDefault("svc_provision.wait", "false")
	viper.SetDefault("svc_provision.timeout", "10m")

	viper.SetDefault("svc_provision.verbose", "false")
	viper.SetDefault("svc_provision.svc_range", "1,100")
}
