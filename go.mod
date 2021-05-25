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
	knative.dev/hack v0.0.0-20210428122153-93ad9129c268
	knative.dev/networking v0.0.0-20210520062317-b2552eca09f7
	knative.dev/pkg v0.0.0-20210525142738-ea766de6bdd1
	knative.dev/serving v0.23.1-0.20210525164648-71e501e141e1
)
