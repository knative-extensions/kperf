# kperf eventing

## Temporary setup

That functionality will be wrapped into `kperf eventing` command in near future.

Currently environment variables are used to configure server side componennts with YAML files.

First copy and setup your environment

```
cp hack/setenv.template.sh hack/setenv.sh
```

Edit `config/setenv.sh` 


## Building eventing support 

See separate docs about [kpef eventing development](./eventing-dev.mg).

## Running performance tests

### kperf prepare

That functionality will be wrapped into `kperf eventing prepare` command in near future.

Currently it needs ti be run from command line to dwpeloy test receiver.

```
source hack/setenv.sh
export REPLICAS=1
cat config/receiver.yaml | envsubst | kubectl apply -f -
```

To start receiver consuming events from Kafka:

```
export KAFKA_BOOTSTRAP_SERVERS=<your Kafka broker>
export KAFKA_TOPIC=<topic>
export KAFKA_GROUP=<consumer group>
```

And to start receiver consuming events from Redis Streams:

```
export REDIS_ADDRESS=<redis server address>
```

### kperf measure

That functionality will be wrapped into `kperf eventing prepare` command in near future.

Currently establising connection for rereiving metrics and deploying driver jobs is done from command line.

#### Create tunnel to listen for metrics

The tunnel is used by `kperf eventing measure` to get metrics from receiver.

```
kubectl -n <namespace> port-forward deployment/kperf-eventing-receiver 8001:8001
```

Verify tunnel works

```
curl http://localhost:8002/metrics
```


#### Start retrieving metrics 

```
./kperf eventing measure
```

Alternatively:

```
go run -mod=readonly ./cmd/ eventing measure
```


#### Start test drviver


Determina location of receiver service in Kubernetes namespace:

```
NS=`kubectl config view --minify --output 'jsonpath={..namespace}'`
export TARGET_URL=http://kperf-eventing-receiver.${NS}.svc.cluster.local
```

The simple version should also work:

```
export TARGET_URL=http://kperf-eventing-receiver
```

Test if everything is setup correctly by running test driver setting events over HTTP to receiver:

```
export RUN_ID=http-baseline2
export CONCURRENT=1
export START=1
export INC=1 
export DURATION=1
export TEST_DURATION=4
cat config/driver-job.yaml | envsubst | kubectl apply -f -
```

Run test driver to directly events to the receiver to determine baseline HTTP performance 

```
export RUN_ID=test4c
export CONCURRENT=10
export START=100
export INC=100 
export DURATION=1
export TEST_DURATION=5
cat config/driver-job.yaml | envsubst | kubectl apply -f -
```

### kperf cleanup

That functionality will be wrapped into `kperf eventing cleanup` command in near future.

Remove receiver deployment and driver jobs from Kubernetes namespace used for testing

```
kubeclt delete deployment kperf-eventing-receiver
kubeclt get jobs --no-headers=true | awk '/kperf-eventing-driver-job/{print $1}' | xargs  kubectl delete job
kubeclt get pods --no-headers=true | awk '/kperf-eventing-driver-job/{print $1}' | xargs  kubectl delete pod
```
