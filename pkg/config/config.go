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
	"os"
	"path/filepath"
	"runtime"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// bootstrapDefaults are the defaults values to use
type defaultConfig struct {
	configFile string
}

// Initialize defaults
var bootstrapDefaults = initDefaults()

//const configContentDefaults = `#
//# service:
//# 	namespace:
//# 	svc-prefix:
//# 	svc-range:
//#	output: /tmp
//# 	create:
//# 		batch:
//# 		concurrency:
//# 	load:
//# 		load-tool:
//# 		load-concurrency:
//# 		load-duration:
//`

// config contains the variables for the kperf config
type config struct {
	// configFile is the config file location
	configFile string
}

// ConfigFile returns the config file which is either the default XDG conform
// config file location or the one set with --config
func (c *config) ConfigFile() string {
	if c.configFile != "" {
		return c.configFile
	}
	return bootstrapDefaults.configFile
}

// Config used for flag binding, available for every sub-command
var globalConfig = config{}

//// GlobalConfig is the global configuration available for every sub-command
//var GlobalConfig = &globalConfig

// BootstrapConfig reads in config file
func BootstrapConfig() error {

	//Create a new FlagSet for the bootstrap flags and parse those. This will
	//initialize the config file to use (obtained via globalConfig.ConfigFile())
	//bootstrapFlagSet := pflag.NewFlagSet("kperf", pflag.ContinueOnError)
	//AddBootstrapFlags(bootstrapFlagSet)
	//bootstrapFlagSet.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	//bootstrapFlagSet.Usage = func() {}
	//err := bootstrapFlagSet.Parse(os.Args)
	//if err != nil && !errors.Is(err, pflag.ErrHelp) {
	//	return err
	//}
	//
	//// Bind flags so that options that have been provided have priority.
	//// Important: Always read options via GlobalConfig methods
	////err = viper.BindPFlag(keyPluginsDirectory, bootstrapFlagSet.Lookup(flagPluginsDir))
	//if err != nil {
	//	return err
	//}

	configFile := globalConfig.ConfigFile()
	viper.SetConfigFile(configFile)
	_, err := os.Lstat(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat configfile %s: %w", configFile, err)
		}
		//if err := os.MkdirAll(filepath.Dir(viper.ConfigFileUsed()), 0775); err != nil {
		//	// Can't create config directory, proceed silently without reading the config
		//	return nil
		//}
		//if err := os.WriteFile(viper.ConfigFileUsed(), []byte(configContentDefaults), 0600); err != nil {
		//	// Can't create config file, proceed silently without reading the config
		//	return nil
		//}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Defaults are taken from the parsed flags, which in turn have bootstrap defaults
	// For now default handling is happening directly in the getter of GlobalConfig
	// viper.SetDefault(keyPluginsDirectory, bootstrapDefaults.pluginsDir)

	// If a config file is found, read it in.
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	return err
}

// AddBootstrapFlags adds bootstrap flags used in a separate bootstrap proceeds
func AddBootstrapFlags(flags *pflag.FlagSet) {
	flags.StringVar(&globalConfig.configFile, "config", "", fmt.Sprintf("kperf configuration file (default: %s)", defaultConfigFileForUsageMessage()))
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

	// Check the deprecated path first and fallback to it, add warning to error message
	if configHome := filepath.Join(home, ".kperf"); dirExists(configHome) {
		migrationPath := filepath.Join(home, ".config", "kperf")
		fmt.Fprintf(os.Stderr, "WARNING: deprecated kperf config directory '%s' detected. Please move your configuration to '%s'\n", configHome, migrationPath)
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
		// Check the deprecated path first and fallback to it, add warning to error message
		if configHome := filepath.Join(home, ".kperf"); dirExists(configHome) {
			migrationPath := filepath.Join(os.Getenv("APPDATA"), "kperf")
			fmt.Fprintf(os.Stderr, "WARNING: deprecated kperf config directory '%s' detected. Please move your configuration to '%s'\n", configHome, migrationPath)
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

// Prepare the default config file for the usage message
func defaultConfigFileForUsageMessage() string {
	if runtime.GOOS == "windows" {
		return "%APPDATA%\\kperf\\config.yaml"
	}
	return "~/.config/kperf/config.yaml"
}
