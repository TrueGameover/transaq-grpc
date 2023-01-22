FROM debian:bookworm-20230109 as base

RUN echo 'deb http://mirror.yandex.ru/debian bookworm main non-free contrib' > /etc/apt/sources.list
RUN apt update && apt install -y --no-install-recommends --no-install-suggests ca-certificates wine64 wine64-tools winetricks && apt clean && update-ca-certificates

RUN mkdir -p /home/server && useradd --uid 1000 --home /home/server server && chown -R 1000:1000 /home/server

ADD dll/txmlconnector64-6.19.2.21.21.dll /app/bin/

EXPOSE 50051


FROM base as prod
ADD bin/server.exe /app/bin/
USER 1000
WORKDIR /app/bin
ENTRYPOINT ["/usr/bin/wine64", "/app/bin/server.exe"]

FROM prod as debug_prod
RUN apt update && apt install -y --no-install-recommends --no-install-suggests wget git && apt clean

RUN cd /tmp && wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && cd /tmp && tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz && rm -f /tmp/go1.18.2.linux-amd64.tar.gz
RUN echo PATH=$PATH:/usr/local/go/bin > /etc/profile

RUN GOOS=windows GOARCH=amd64 /usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest

ADD bin/server.exe /app/bin/
WORKDIR /app/bin
EXPOSE 2345

ENTRYPOINT ["/usr/bin/wine64 /go/bin/windows_amd64/dlv.exe --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec /app/bin/server.exe"]
#ENTRYPOINT ["/usr/bin/wine64", "start", "/unix", "/go/bin/windows_amd64/dlv.exe", "\"--listen=:2345 --headless=true --api-version=2 --accept-multiclient exec /q $(winepath --windows app/bin/server.exe)\""]