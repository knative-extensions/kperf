#!/bin/bash

# Copyright 2020 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export VERSION=0.1
export IMAGE_NAME=${DOCKER_REGISTRY:-docker.io/aslom}/kperf:${VERSION}
echo IMAGE_NAME=$IMAGE_NAME
export RUN_ID=http-baseline1
export START=100
export INC=100 
export DURATION=1
export TEST_DURATION=1
# For local setup
#export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
#export KAFKA_TOPIC=topic10
#export KAFKA_GROUP=test4
#export REDIS_ADDRESS=localhost:6379
# For cloud setup
export KAFKA_BOOTSTRAP_SERVERS=
export KAFKA_TOPIC=
export KAFKA_GROUP=
export REDIS_ADDRESS=
