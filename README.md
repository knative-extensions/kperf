# kperf

[![codecov](https://codecov.io/gh/zyjiaobj/kperf/branch/master/graph/badge.svg?token=N77G9OJIBA)](https://codecov.io/gh/zyjiaobj/kperf)

**[This component is ALPHA](https://github.com/knative/community/tree/main/mechanics/MATURITY-LEVELS.md)**

A Knative Load Test Tool

**Please NOTE this project is under rapid development**.
Kperf is designed for Knative Load test. It helps to generate workload like Knative services and
gives measurement result about underneath resource create time duration based on server side timestamps,
and give statics and raw measurement result, to help Knative developers or operators to figure out Knative platform
stability, scalability and performance bottleneck.


## Build and install kperf
```cassandraql
# format and build kperf
$ cd {workspace}/src/knative.dev/kperf
$ ./hack/build.sh

# Move kperf
$ mv kperf /usr/local/bin/
```

## Using Kperf

```bash
$ kperf service --help
Knative service load test and measurement. For example:

kperf service measure --range 1,10, --name perf - to measure 10 Knative service named perf-x in perf-x namespace

Usage:
  kperf service [command]

Available Commands:
  clean       clean ksvc
  generate    generate Knative Service
  help        Help about any command
  load        Load and Measure Knative service
  measure     Measure Knative service
  scale       Scale and Measure Knative service

Flags:
  -h, --help   help for service

Use "kperf service [command] --help" for more information about a command.
```

See the [docs](docs/examples.md) for more details on how to run the individual commands.

