# kperf
A Knative Load Test Tool

Please **NOTE** this project is under development.
Kperf is designed for Knative Load test. It helps to generate workload like Knative services and 
gives measurement result about underneath resource create time duration based on server side timestamps, 
and give statics and raw measurement result, to help Knative developers or operators to figure out Kntive platform
stability, scalability and performance bottleneck.


# Usage
## Build kperf
```cassandraql
# format and build kperf
$ cd {workspace}/src/github.com/zhanggbj/kperf
$ ./hack/build.sh

# Move kperf
$ mv kperf /usr/local/bin/
```

Note: [go-bindata](https://github.com/go-bindata/go-bindata) is required in the build process.

## User Stories

### As a Codeengine developer, I want to generate Codeengine Service concurrently for test
```
# Generate total 30 knative service, for each 15 seconds create 10 ksvc with 5 concurrency in namespace test1, test2 and test3, and the ksvc names are ktest1, ktest2.....ktest29.
$ kperf service generate -n 30 -b 10 -c 5 -i 15 --nsPrefix test --nsRange 1,3 --svcPrefix ktest --maxScale 1 --minScale 1

Creating ksvc ktestsss0 in namespace test1
Creating ksvc ktestsss1 in namespace test2
...
...
Creating ksvc ktestsss29 in namespace test3
```
```
# Generate total 30 knative service, for each 15 seconds create 10 ksvc with 1 concurrency in namespace test1, test2 and test3, and the ksvc names are ktest1, ktest2.....ktest29. The generation will wait the previous genreated service to be ready for most 10 seconds.
$ kperf service generate -n 30 -b 10 -c 5 -i 15 --nsPrefix test --nsRange 1,3 --svcPrefix ktest --wait --timeout 10s --maxScale 1 --minScale 1

Creating ksvc ktestsss0 in namespace test1
Creating ksvc ktestsss1 in namespace test2
...
...
Creating ksvc ktestsss29 in namespace test3
```

### As a Codeengine developer, I want to clean Codeengine Service generated for test
```
# Delete all ksvc with name prefix ktest in namespaces with name prefix test and index 1,2,3
$ kperf service clean --nsPrefix test --nsRange 1,3 --svcPrefix ktest

Delete ksvc ktestsss0 in namespace test1
Delete ksvc ktestsss12 in namespace test1
...
Delete ksvc ktestsss1 in namespace test2
Delete ksvc ktestsss10 in namespace test2
...
Delete ksvc ktestsss5 in namespace test3
Delete ksvc ktestsss8 in namespace test3
```

### As a Codeengine developer, I want to measure Codeengine Service creation time duration
- Service Configurations Duration Measurement: time duration for Knative Configurations to be ready
- Service Routes Duration Measurement: time duration for Knative Routes to be ready
- Overall Service Ready Measurement: time duration for Knative Service to be ready
![resources created by Knative Service](docs/service_creation.png)

**Example 1 Measure Services (for eg. range 1,500)for load test under a specific namespace**

```cassandraql
$ kperf service measure -n ktest-1 --prefix ktest --range 101,110 --verbose
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
**Sample Dashboard**
![service_creation_duration measurement](docs/kperf_dashboard.png)





