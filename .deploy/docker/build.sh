#!/bin/bash

# shellcheck disable=SC2046
export $(grep -v '^#' .env | xargs -d '\n')

docker build -f .deploy/docker/build/Dockerfile -t "$DOCKER_IMAGE_REGISTRY/$DOCKER_IMAGE_REPOSITORY/transaq-grpc-build:$DOCKER_IMAGE_TAG" . || exit 1
docker run -t --volume="$(pwd)/bin:/go/src/github.com/TrueGameover/transaq-grpc/bin" "$DOCKER_IMAGE_REGISTRY/$DOCKER_IMAGE_REPOSITORY/transaq-grpc-build:$DOCKER_IMAGE_TAG" || exit 1

docker build -f .deploy/docker/app/Dockerfile --build-arg="DOCKER_USER_ID=$DOCKER_USER" --target=prod -t "$DOCKER_IMAGE_REGISTRY/$DOCKER_IMAGE_REPOSITORY/transaq-grpc:$DOCKER_IMAGE_TAG" . || exit 1
docker push "$DOCKER_IMAGE_REGISTRY/$DOCKER_IMAGE_REPOSITORY/transaq-grpc:$DOCKER_IMAGE_TAG" || exit 1
