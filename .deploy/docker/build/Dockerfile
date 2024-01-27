FROM golang:1.21-bookworm as builder

RUN echo 'deb http://mirror.yandex.ru/debian bookworm main non-free contrib' > /etc/apt/sources.list
RUN apt update && apt install -y gcc-mingw-w64 gcc libc-dev musl-dev

RUN mkdir /.cache && mkdir /app && chown -R 1000:1000 /go /.cache /app
USER 1000

ADD go.mod /go/src/github.com/TrueGameover/transaq-grpc/
ADD go.sum /go/src/github.com/TrueGameover/transaq-grpc/
WORKDIR /go/src/github.com/TrueGameover/transaq-grpc
RUN go mod download

ADD src/ /go/src/github.com/TrueGameover/transaq-grpc/src/

ENV CGO_ENABLED=1
ENV CC="x86_64-w64-mingw32-gcc"
ENV CXX="x86_64-w64-mingw32-g++"
ENV GOOS=windows
ENV GOARCH=amd64

RUN go build -ldflags "-extldflags -static" -o /app/server.exe ./src/main.go

ENTRYPOINT ["cp", "/app/server.exe", "./bin/"]