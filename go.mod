module knative.dev/kperf

go 1.14

require (
	bou.ke/monkey v1.0.2
	github.com/kevinburke/go-bindata v3.22.0+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/montanaflynn/stats v0.6.5
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	knative.dev/hack v0.0.0-20210622141627-e28525d8d260
	knative.dev/networking v0.0.0-20210622182128-53f45d6d2cfa
	knative.dev/pkg v0.0.0-20210622173328-dd0db4b05c80
	knative.dev/serving v0.23.1-0.20210623204144-c54eeabbb984
)
