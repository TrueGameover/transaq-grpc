.DEFAULT_GOAL := ps

ps:
	@docker-compose ps

log:
	@docker-compose logs --tail=100 app

stop:
	@docker-compose stop app

start:
	@docker-compose up -d --build app

stop-debug:
	@docker-compose stop debug

start-debug:
	@docker-compose up --build debug

log-debug:
	@docker-compose logs --tail=100 debug

docker-up:
	@docker-compose up -d --build

docker-down:
	@docker-compose down

build:
	@docker-compose up --build build

build-and-run:
	make build
	@docker-compose up --build app

build-and-debug:
	make build
	@docker-compose up --build debug

build-proto:
	# local
	@protoc --proto_path=proto \
  --go_out=./src/grpc --go_opt=Mconnect.proto=/server \
  --go-grpc_out=./src/grpc --go-grpc_opt=Mconnect.proto=/server \
  proto/connect.proto

