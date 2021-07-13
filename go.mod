module knative.dev/kperf

go 1.14

require (
	bou.ke/monkey v1.0.2
	github.com/kevinburke/go-bindata v3.22.0+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/montanaflynn/stats v0.6.5
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	knative.dev/hack v0.0.0-20210622141627-e28525d8d260
	knative.dev/networking v0.0.0-20210713052150-e937f69e4529
	knative.dev/pkg v0.0.0-20210712150822-e8973c6acbf7
	knative.dev/serving v0.24.1-0.20210713092349-05af7d4acdb7
)
