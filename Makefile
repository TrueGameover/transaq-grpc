.DEFAULT_GOAL := ps

build-proto:
	# local
	@protoc --proto_path=proto \
  --go_out=./src/grpc --go_opt=Mconnect.proto=/server \
  --go-grpc_out=./src/grpc --go-grpc_opt=Mconnect.proto=/server \
  proto/connect.proto

push:
	@make build
	@make start
	@docker-compose push app
	@docker-compose down

minikube-build:
	./.deploy/docker/build.sh minikube

minikube-update-k8s:
	./.deploy/k8s/secrets.sh && ./.deploy/k8s/apply.sh

build:
	./.deploy/docker/build.sh minikube
