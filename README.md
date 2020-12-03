# kperf

[![codecov](https://codecov.io/gh/zyjiaobj/kperf/branch/master/graph/badge.svg?token=N77G9OJIBA)](https://codecov.io/gh/zyjiaobj/kperf)

A Knative Load Test Tool

**Please NOTE this project is under rapid development**.
Kperf is designed for Knative Load test. It helps to generate workload like Knative services and 
gives measurement result about underneath resource create time duration based on server side timestamps, 
and give statics and raw measurement result, to help Knative developers or operators to figure out Knative platform
stability, scalability and performance bottleneck.


# Usage
## Build and install kperf
```cassandraql
# format and build kperf
$ cd {workspace}/src/knative.dev/kperf
$ go get -u github.com/jteeuwen/go-bindata/...
$ export PATH=$PATH:$GOPATH/bin
$ ./hack/build.sh

# Move kperf
$ mv kperf /usr/local/bin/
```

Note: [go-bindata](https://github.com/go-bindata/go-bindata) is required in the build process.

## Knative Serving load test

Kperf can help to generate Knative Service Deployment Load in your Knative platform. We assume you have created a
Kubernetes cluster and deployed [Knative Serving](https://knative.dev/docs/install/). 

### Prepare namespaces
Please note that by default kperf assumes you have prepared K8s namespace(s) to create Knative Service. 
If namespace doesn't exist, create it with kubectl as below

```shell script
# Create a namespace for kperf to create Services in it
kubectl create ns {namespace-name}

# Create namespaces from test-1 to test-10 for kperf to create Services distributed in them
for name in {1..10};do kubectl create ns test-$name;done
```

### generate Knative Service deployment load
```shell script
# Generate total 30 knative service, for each 15 seconds create 10 ksvc with 5 concurrency in namespace test-1, test-2
# and test-33, and the ksvc names are ktest-0, ktest-1.....ktest-29.
$ kperf service generate -n 30 -b 10 -c 5 -i 15 --namespace-prefix test --namespace-range 1,3 --svc-prefix ktest --max-scale 1 --min-scale 1

Creating ksvc ktest-0 in namespace test-1
Creating ksvc ktest-1 in namespace test-2
...
...
Creating ksvc ktest-29 in namespace test-3
```

```shell script
# Generate total 30 knative service, for each 15 seconds create 10 ksvc with 1 concurrency in namespace test1, test2 and
# test3, and the ksvc names are ktest-0, ktest-2.....ktest-29. The generation will wait the previous generated service
# to be ready for most 10 seconds.
$ kperf service generate -n 30 -b 10 -c 5 -i 15 --namespace-prefix test --namespace-range 1,3 --svc-prefix ktest --wait --timeout 10s --max-scale 1 --min-scale 1

Creating ksvc ktests-0 in namespace test-1
Creating ksvc ktests-1 in namespace tes-2
...
...
Creating ksvc ktests-29 in namespace test-3
```

### Measure Knative Service deployment time
- Service Configurations Duration Measurement: time duration for Knative Configurations to be ready
- Service Routes Duration Measurement: time duration for Knative Routes to be ready
- Overall Service Ready Measurement: time duration for Knative Service to be ready

Here is a figure of different resources generated for a Knative Service(assuming Istio is the network solution).
![resources created by Knative Service](docs/service_creation.png)

**Example 1 Measure Services (for eg. range 1,500)for load test under a specific namespace**

```shell script
$ kperf service measure -n ktest-1 --svc-prefix ktest --range 101,110 --verbose
[Verbose] Service ktest-101: Service Configuration Ready Duration is 5s/5.000000s
[Verbose] Service ktest-101: - Service Revision Ready Duration is 5s/5.000000s
[Verbose] Service ktest-101:   - Service Deployment Created Duration is 2s/2.000000s
[Verbose] Service ktest-101:     - Service Pod Scheduled Duration is 0s/0.000000s
[Verbose] Service ktest-101:     - Service Pod Containers Ready Duration is 0s/0.000000s
[Verbose] Service ktest-101:       - Service Pod queue-proxy Started Duration is 0s/0.000000s
[Verbose] Service ktest-101:       - Service Pod user-container Started Duration is 0s/0.000000s
[Verbose] Service ktest-101: Service Route Ready Duration is 7s/7.000000s
[Verbose] Service ktest-101: - Service Ingress Ready Duration is 2s/2.000000s
[Verbose] Service ktest-101:   - Service Ingress Network Configured Duration is 0s/0.000000s
[Verbose] Service ktest-101:   - Service Ingress LoadBalancer Ready Duration is 2s/2.000000s
[Verbose] Service ktest-101: Overall Service Ready Duration is 7s/7.000000s
-------- Measurement --------
Ready: 10 Fail: 0
Service Configuration Duration:
Total: 138.000000s
Average: 13.800000s
- Service Revision Duration:
  Total: 135.000000s
  Average: 13.500000s
  - Service Deployment Created Duration:
    Total: 135.000000s
    Average: 13.500000s
    - Service Pod Scheduled Duration:
      Total: 0.000000s
      Average: 0.000000s
    - Service Pod Containers Ready Duration:
      Total: 0.000000s
      Average: 0.000000s
      - Service Pod queue-proxy Started Duration:
        Total: 0.000000s
        Average: 0.000000s
      - Service Pod user-container Started Duration:
        Total: 0.000000s
        Average: 0.000000s

Service Route Ready Duration:
Total: 167.000000s
Average: 16.700000s
- Service Ingress Ready Duration:
  Total: 26.000000s
  Average: 2.600000s
  - Service Ingress Network Configured Duration:
    Total: 0.000000s
    Average: 0.000000s
  - Service Ingress LoadBalancer Ready Duration:
    Total: 26.000000s
    Average: 2.600000s

-----------------------------
Overall Service Ready Measurement:
Ready: 10 Fail: 0
Total: 167.000000s
Average: 16.700000s
Measurement saved in CSV file /tmp/20200710182129_ksvc_creation_time.csv
Visualized measurement saved in HTML file /tmp/20200710182129_ksvc_creation_time.html

$ cat /tmp/20200710182129_ksvc_creation_time.csv                                         
  svc_name,svc_namespace,configuration_ready,revision_ready,deployment_created,pod_scheduled,containers_ready,queue-proxy_started,user-container_started,route_ready,ingress_ready,ingress_config_ready,ingress_lb_ready,overall_ready
  ktest-101,ktest-1,5,5,2,0,0,0,0,7,2,0,2,7
  ktest-102,ktest-1,8,8,2,0,0,0,0,10,2,0,2,10
  ktest-103,ktest-1,43,42,1,0,0,0,0,47,3,0,3,47
  ktest-104,ktest-1,16,16,3,0,0,0,0,18,2,0,2,18
  ktest-105,ktest-1,8,7,1,0,0,0,0,10,2,0,2,10
  ktest-106,ktest-1,8,8,2,0,0,0,0,11,3,0,3,11
  ktest-107,ktest-1,12,12,1,0,0,0,0,16,3,0,3,16
  ktest-108,ktest-1,11,11,2,0,0,0,0,15,4,0,4,15
  ktest-109,ktest-1,13,13,2,0,0,0,0,15,2,0,2,15
  ktest-110,ktest-1,14,13,1,0,0,0,0,18,3,0,3,18
```

### Clean Knative Service generated for test
```shell script
# Delete all ksvc with name prefix ktest in namespaces with name prefix test and index 1,2,3
$ kperf service clean --namespace-prefix test --namespace-range 1,3 --svc-prefix ktest

Delete ksvc ktest-0 in namespace test-1
Delete ksvc ktest-2 in namespace test-1
...
Delete ksvc ktests-1 in namespace test-2
Delete ksvc ktests-10 in namespace test-2
...
Delete ksvc ktests-5 in namespace test-3
Delete ksvc ktests-8 in namespace test-3
```

**Sample Dashboard**
![service_creation_duration measurement](docs/kperf_dashboard.png)





