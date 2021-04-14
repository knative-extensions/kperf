export VERSION=0.1
export IMAGE_NAME=${DOCKER_REGISTRY:-docker.io/aslom}/kperf:${VERSION}
echo IMAGE_NAME=$IMAGE_NAME
export RUN_ID=http-baseline1
export START=100
export INC=100 
export DURATION=1
export TEST_DURATION=1
