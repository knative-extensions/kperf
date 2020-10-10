package main

import (
	"fmt"
	"os"

	"github.com/zhanggbj/kperf/core"
)

func main() {
	if err := core.NewPerfCommand().Execute(); err != nil {
		fmt.Println("failed to execute kperf command:", err)
		os.Exit(1)
	}
}
