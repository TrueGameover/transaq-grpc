FROM debian:bookworm-20240110 as base

RUN echo 'deb http://mirror.yandex.ru/debian bookworm main non-free contrib' > /etc/apt/sources.list
RUN apt update && apt install -y --no-install-recommends wget ca-certificates htop && apt clean && update-ca-certificates

RUN wget -O /etc/apt/keyrings/winehq-archive.key https://dl.winehq.org/wine-builds/winehq.key
RUN wget -NP /etc/apt/sources.list.d/ https://dl.winehq.org/wine-builds/debian/dists/bookworm/winehq-bookworm.sources

RUN dpkg --add-architecture i386

RUN apt update && \
    apt install -y --no-install-recommends winehq-stable \
    && apt clean

RUN mkdir -p /home/server && useradd --uid 1000 --home /home/server --shell /bin/bash server \
    && chown -R 1000:1000 /home/server \
    && mkdir -p /app/bin/logs && chmod 0777 /app/bin/logs \
    && mkdir -p /app/bin/certs && chmod 0777 /app/bin/certs

ADD .deploy/docker/certs/ /app/bin/certs/
RUN chmod -R 0777 /app/bin/certs

ADD dll/txmlconnector64-6.19.2.21.21.dll /app/bin/

EXPOSE 50051


FROM base as prod
ADD .deploy/docker/app/entrypoint.sh /app/
ADD bin/server.exe /app/bin/
USER 1000
WORKDIR /app/bin
ENTRYPOINT ["/app/entrypoint.sh"]

FROM alpine:3.17.1 AS final

COPY --from=prod /app/ /app/
ENTRYPOINT ["/app/entrypoint.sh"]

FROM base as debug_prod
RUN apt update && apt install -y --no-install-recommends --no-install-suggests wget git && apt clean

RUN cd /tmp && wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && cd /tmp && tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz && rm -f /tmp/go1.18.2.linux-amd64.tar.gz
RUN echo PATH=$PATH:/usr/local/go/bin > /etc/profile

USER 1000

RUN GOOS=windows GOARCH=amd64 /usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest
ADD docker/app/entrypoint_debug.sh /app
ADD bin/server.exe /app/bin/

WORKDIR /app/bin
EXPOSE 2345

ENTRYPOINT ["/app/entrypoint_debug.sh"]
