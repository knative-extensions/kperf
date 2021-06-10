module knative.dev/kperf

go 1.14

require (
	bou.ke/monkey v1.0.2
	github.com/kevinburke/go-bindata v3.22.0+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/montanaflynn/stats v0.6.5
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	knative.dev/hack v0.0.0-20210609124042-e35bcb8f21ec
	knative.dev/networking v0.0.0-20210610043142-ddb4035f00e9
	knative.dev/pkg v0.0.0-20210610083643-00fa1549f723
	knative.dev/serving v0.23.1-0.20210609225242-ea552e33d1fc
)
