#!/bin/bash

envFile=".env.prod"

if [ "$1" = "minikube" ]; then
  envFile=".env"
fi

# shellcheck disable=SC2046
# shellcheck disable=SC2016
export $(envsubst '$HOME' <$envFile | grep -v '^#' | xargs -d '\n')

# shellcheck disable=SC2016
replacedEnvs='${IMAGE_PULL_SEC} ${DOCKER_IMAGE_REGISTRY} ${DOCKER_IMAGE_REPOSITORY} ${DOCKER_IMAGE_TAG}'

envsubst "$replacedEnvs" <.deploy/k8s/deployment.k8s.yml | kubectl --kubeconfig="$K8S_CONFIG_PATH" apply -n "$NAMESPACE" -f - || exit 1
