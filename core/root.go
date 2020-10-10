package core

import (
	"fmt"
	"github.com/zhanggbj/kperf/pkg/command/service"
	"github.com/zhanggbj/kperf/pkg/command/version"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zhanggbj/kperf/pkg"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands

func NewPerfCommand(params ...pkg.PerfParams) *cobra.Command {
	p := &pkg.PerfParams{}
	p.Initialize()

	rootCmd := &cobra.Command{
		Use:   "kperf",
		Short: "A CLI of to help with Knative performance test",
		Long:  `A CLI of to help with Knative performance test.`,
	}
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/kubectl)")
	rootCmd.AddCommand(service.NewServiceCmd(p))
	rootCmd.AddCommand(version.NewVersionCommand())
	rootCmd.InitDefaultHelpCmd()
	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".kperf")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
