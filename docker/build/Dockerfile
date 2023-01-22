FROM golang:1.18.2-bullseye as builder

RUN echo 'deb http://mirror.yandex.ru/debian bullseye main non-free contrib' > /etc/apt/sources.list
RUN apt update && apt install -y gcc-mingw-w64 gcc libc-dev musl-dev

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

RUN CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" \
    CXX="x86_64-w64-mingw32-g++" GOOS=windows GOARCH=amd64  \
    go build -ldflags "-extldflags -static" -o ./server.exe ./src/main.go

RUN chown 1000:1000 ./server.exe

ENTRYPOINT ["cp", "./server.exe", "./bin/"]