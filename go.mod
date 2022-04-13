module knative.dev/kperf

go 1.16

require (
	bou.ke/monkey v1.0.2
	github.com/google/go-containerregistry v0.8.1-0.20220219142810-1571d7fdc46e // indirect
	github.com/kevinburke/go-bindata v3.23.0+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/montanaflynn/stats v0.6.5
	github.com/spf13/afero v1.8.0 // indirect
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	gotest.tools/v3 v3.1.0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	knative.dev/hack v0.0.0-20220411131823-6ffd8417de7c
	knative.dev/networking v0.0.0-20220412163509-1145ec58c8be
	knative.dev/pkg v0.0.0-20220412134708-e325df66cb51
	knative.dev/serving v0.30.1-0.20220413003907-2de1474b55ba
)
