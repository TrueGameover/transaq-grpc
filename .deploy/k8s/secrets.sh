#!/bin/bash

# shellcheck disable=SC2046
export $(grep -v '^#' .env | xargs -d '\n')

kubectl -n "$NAMESPACE" delete --ignore-not-found=true secret transaq-grpc-secrets &&
  kubectl -n "$NAMESPACE" create secret generic transaq-grpc-secrets --from-env-file=.env
