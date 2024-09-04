#!/bin/bash

envFile=".env.prod"

if [ "$1" = "minikube" ]; then
  envFile=".env"
fi

# shellcheck disable=SC2046
# shellcheck disable=SC2016
export $(envsubst '$HOME' <$envFile | grep -v '^#' | xargs -d '\n')

kubectl --kubeconfig="$K8S_CONFIG_PATH" -n "$NAMESPACE" delete --ignore-not-found=true secret transaq-grpc-secrets || exit 1
kubectl --kubeconfig="$K8S_CONFIG_PATH" -n "$NAMESPACE" create secret generic transaq-grpc-secrets --from-env-file=.env || exit 1
