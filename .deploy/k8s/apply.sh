#!/bin/bash

# shellcheck disable=SC2046
export $(grep -v '^#' .env | xargs -d '\n')

# shellcheck disable=SC2016
replacedEnvs='${IMAGE_PULL_SEC} ${DOCKER_IMAGE_REGISTRY} ${DOCKER_IMAGE_REPOSITORY} ${DOCKER_IMAGE_TAG}'

envsubst "$replacedEnvs" <.deploy/k8s/deployment.k8s.yml | kubectl apply -n "$NAMESPACE" -f - || exit 1
