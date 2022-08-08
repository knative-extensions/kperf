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
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type defaultConfig struct {
	configFile string
}

var serviceCommonFlagsSet = map[string]bool{
	"namespace":        true,
	"namespace-prefix": true,
	"namespace-range":  true,
	"svc-prefix":       true,
	"range":            true,
	"output":           true,
}

// Initialize common flags in config file
var commonFlags = initCommonConfig()

// Initialize defaults, bootstrapDefaults are the defaults values to use
var bootstrapDefaults = initDefaults()

// Config contains the variables for the kperf config
type config struct {
	// ConfigFile is the config file location
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

// BootstrapConfig reads in config file
// if config file is set, read into viper
// if config file location is not set, read from default config file location,
// while default config file doesn't exist, create a nil file
func BootstrapConfig() error {
	configFile := globalConfig.ConfigFile()
	viper.SetConfigFile(configFile)
	_, err := os.Lstat(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat configfile %s: %w", configFile, err)
		}
		// If file or directory doesn't exist, create it
		if err := os.MkdirAll(filepath.Dir(viper.ConfigFileUsed()), 0775); err != nil {
			// Can't create config directory, proceed silently without reading the config
			log.Println("Can't create config directory, proceed silently without reading the config")
			return nil
		}
		if err := os.WriteFile(viper.ConfigFileUsed(), []byte(""), 0600); err != nil {
			// Can't create config file, proceed silently without reading the config
			log.Println("Can't create config file, proceed silently without reading the config")
			return nil
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read into viper
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}
	return nil
}

// AddBootstrapFlags adds bootstrap flags used in a separate bootstrap proceeds
func AddBootstrapFlags(flags *pflag.FlagSet) {
	flags.StringVar(&globalConfig.configFile, "config", defaultConfigLocation(), "kperf configuration file")
}

// Initialize defaults. This happens lazily go allow to change the
// home directory for e.g. tests
func initDefaults() *defaultConfig {
	return &defaultConfig{
		configFile: defaultConfigLocation(),
	}
}

func defaultConfigLocation() string {
	var base string
	if runtime.GOOS == "windows" {
		base = defaultConfigDirWindows()
	} else {
		base = defaultConfigDirUnix()
	}
	return filepath.Join(base, "config.yaml")
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

// BindFlags binds each cobra flag to its associated viper configuration (config file and environment variable),
// and validate required flag(s)
func BindFlags(cmd *cobra.Command, configPrefix string, set map[string]bool) (err error) {
	keys := make([]string, 0)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			fmt.Println("#", f.Name, "is not set in command, setFlagFromConfig")
			err = setFlagFromConfig(cmd.Flags(), f, configPrefix, set, &keys)
			if err != nil {
				return
			}
		} else {
			fmt.Println("#", f.Name, " is set in command")
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		// if !f.Changed {
		// 	prefixArray := strings.Split(configPrefix, ".")
		// 	// Search from common flags firstly
		// 	parentCommand := prefixArray[0]
		// 	if viper.IsSet(configPrefix + f.Name) {
		// 		val := viper.Get(configPrefix + f.Name)
		// 		err = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		// 		if err != nil {
		// 			return
		// 		}
		// 	} else if viper.IsSet(parentCommand + f.Name) {
		// 		val := viper.Get(parentCommand + f.Name)
		// 		err = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		// 		if err != nil {
		// 			return
		// 		}
		// 	} else if set != nil {
		// 		// Validate required flags
		// 		if set[f.Name] {
		// 			set[f.Name] = false
		// 			keys = append(keys, f.Name)
		// 		}
		// 	}
		// }
	})
	if err != nil {
		return err
	}
	return validateRequiredFlags(set, keys)
	// Return error on required flag(s)
	// var m string
	// for _, k := range keys {
	// 	if !set[k] {
	// 		if m == "" {
	// 			m = "failed to get required flags, required flag(s) \"" + k + "\""
	// 		} else {
	// 			m = m + ", \"" + k + "\""
	// 		}
	// 		// Reset the value
	// 		set[k] = true
	// 	}
	// }
	// if m != "" {
	// 	err = fmt.Errorf(m)
	// 	return err
	// }
	// return nil
}

// Initialize common flags in config file
// TODO: add common flags for eventing subcommands
func initCommonConfig() map[string]map[string]bool {
	common := make(map[string]map[string]bool)
	common["service"] = serviceCommonFlagsSet
	return common
}

// setFlagsByConfig applies viper config value to flag when flag is not set in command and config has a value
func setFlagFromConfig(flagSet *pflag.FlagSet, f *pflag.Flag, configPrefix string, set map[string]bool, keys *[]string) error {
	commonPrefix := strings.Split(configPrefix, ".")[0]
	// Precedure:flag value in configPrefix > flag value in parent common
	var val interface{}
	if viper.IsSet(configPrefix + f.Name) { // flag value in flags
		val = viper.Get(configPrefix + f.Name)
		fmt.Println("	find in config prefix, val = ", val)
	} else if viper.IsSet(commonPrefix + "." + f.Name) { // flag value in common flags
		// TODO: get nil value
		val = viper.Get(commonPrefix + f.Name)
		fmt.Println("	find in common, val = ", val)
	} else if set != nil {
		// Validate required flags
		fmt.Println(f.Name, " not find in config")
		if set[f.Name] {
			set[f.Name] = false
			// *keys = append(*keys, f.Name) // a stable order to iterate a map for print err info
		}
		return nil
	}

	if val != nil {
		err := flagSet.Set(f.Name, fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
	}

	return nil
}

//	// Return error on required flag(s)
func validateRequiredFlags(set map[string]bool, keys []string) error {
	var m string
	log.Printf("%v\n", set)
	for k, v := range set {
		if !v {
			log.Println("required flag(s) - " + k)
			if m == "" {
				m = "failed to get required flags, required flag(s) \"" + k + "\""
			} else {
				m = m + ", \"" + k + "\""
			}
			// Reset the value
			set[k] = true
		}
		// for _, k := range keys {
		// 	if !set[k] {
		// 		if m == "" {
		// 			m = "failed to get required flags, required flag(s) \"" + k + "\""
		// 		} else {
		// 			m = m + ", \"" + k + "\""
		// 		}
		// 		// Reset the value
		// 		set[k] = true
		// 	}
	}
	if m != "" {
		return fmt.Errorf(m)
	}
	return nil
}
