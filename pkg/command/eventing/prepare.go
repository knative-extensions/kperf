// Copyright 2020 The Knative Authors
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

package eventing

import (
	"errors"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/spf13/cobra"

	"knative.dev/kperf/pkg"
)

func NewEventingPrepareCommand(p *pkg.PerfParams) *cobra.Command {
	generateArgs := generateArgs{}

	ksvcGenCommand := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare Knative Eventing receiver",
		Long: `Prepare Knative Eventing receiver
For example:
# To prepare Knative Eventing receiver
kperf eventing prepare --namespace-prefix testns/ --namespace nsname)
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			if flags.Changed("namespace-prefix") && flags.Changed("namespace") {
				return errors.New("expected either namespace with prefix & range or only namespace name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			var t *template.Template
			var err error

			// load template; if an argument is not specified, default to stdin
			t, err = parseFiles("./config/receiver.gotemplate.yaml")
			die(err)
			// TODO read form stdin "-"
			// bytes, err := ioutil.ReadAll(os.Stdin)
			// die(err)
			// t, err = parse(string(bytes))
			// die(err)

			// get environment variables to supply to the template
			env := readEnv()

			// get writer for rendered output; if an output file is not
			// specified, default to stdout
			//var w io.Writer
			w := os.Stdout

			// render the template
			err = t.Execute(w, env)
			die(err)
			return nil
		},
	}
	// ksvcGenCommand.Flags().IntVarP(&generateArgs.number, "number", "n", 0, "Total number of Knative Service to be created")
	// ksvcGenCommand.MarkFlagRequired("number")
	// ksvcGenCommand.Flags().IntVarP(&generateArgs.interval, "interval", "i", 0, "Interval for each batch generation")
	// ksvcGenCommand.MarkFlagRequired("interval")
	// ksvcGenCommand.Flags().IntVarP(&generateArgs.batch, "batch", "b", 0, "Number of Knative Service each time to be created")
	// ksvcGenCommand.MarkFlagRequired("batch")
	// ksvcGenCommand.Flags().IntVarP(&generateArgs.concurrency, "concurrency", "c", 10, "Number of multiple Knative Services to make at a time")
	// ksvcGenCommand.Flags().IntVarP(&generateArgs.minScale, "min-scale", "", 0, "For autoscaling.knative.dev/minScale")
	// ksvcGenCommand.Flags().IntVarP(&generateArgs.maxScale, "max-scale", "", 0, "For autoscaling.knative.dev/minScale")

	ksvcGenCommand.Flags().StringVarP(&generateArgs.namespacePrefix, "namespace-prefix", "", "", "Namespace prefix. The Knative Services will be created in the namespaces with the prefix")
	ksvcGenCommand.Flags().StringVarP(&generateArgs.namespaceRange, "namespace-range", "", "", "")
	ksvcGenCommand.Flags().StringVarP(&generateArgs.namespace, "namespace", "", "", "Namespace name. The Knative Services will be created in the namespace")

	// ksvcGenCommand.Flags().StringVarP(&generateArgs.svcPrefix, "svc-prefix", "", "ksvc", "Knative Service name prefix. The Knative Services will be ksvc-1,ksvc-2,ksvc-3 and etc.")
	// ksvcGenCommand.Flags().BoolVarP(&generateArgs.checkReady, "wait", "", false, "Whether to wait the previous Knative Service to be ready")
	// ksvcGenCommand.Flags().DurationVarP(&generateArgs.timeout, "timeout", "", 10*time.Minute, "Duration to wait for previous Knative Service to be ready")

	return ksvcGenCommand
}

// based on https://github.com/subfuzion/envtpl/blob/master/cmd/envtpl/main.go

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// returns a new template with custom function maps
// (sprig and environment key prefix matcher) applied
func parseFiles(files ...string) (*template.Template, error) {
	//return template.New(filepath.Base(files[0])).Funcs(sprig.TxtFuncMap()).Funcs(customFuncMap()).ParseFiles(files...)
	return template.New(filepath.Base(files[0])).Funcs(customFuncMap()).ParseFiles(files...)
}

// returns map of environment variables
func readEnv() (env map[string]string) {
	env = make(map[string]string)
	for _, setting := range os.Environ() {
		pair := strings.SplitN(setting, "=", 2)
		env[pair[0]] = pair[1]
	}
	return
}

// custom function that returns key, value for all environment variable keys matching prefix
// (see original envtpl: https://pypi.org/project/envtpl/)
func environment(prefix string) map[string]string {
	env := make(map[string]string)
	for _, setting := range os.Environ() {
		pair := strings.SplitN(setting, "=", 2)
		if strings.HasPrefix(pair[0], prefix) {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

// returns custom template functions map
func customFuncMap() template.FuncMap {
	var functionMap = map[string]interface{}{
		"environment": environment,
	}
	return template.FuncMap(functionMap)
}
