package service

import (
	"github.com/spf13/cobra"
	"github.com/zhanggbj/kperf/pkg"
)

// domainCmd represents the domain command
func NewServiceCmd(p *pkg.PerfParams) *cobra.Command {
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Knative service load test",
		Long: `Knative service load test and measurement. For example:

kperf service measurement --range 1,10, --name perf - to measure 10 Knative service named perf-x in perf-x namespace`,
	}
	serviceCmd.AddCommand(NewServiceMeasureCommand(p))
	serviceCmd.AddCommand(NewServiceGenerateCommand(p))
	serviceCmd.AddCommand(NewServiceCleanCommand(p))

	serviceCmd.InitDefaultHelpCmd()
	return serviceCmd
}
