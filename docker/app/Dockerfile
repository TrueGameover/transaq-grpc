FROM debian:bookworm as base

RUN echo 'deb http://mirror.yandex.ru/debian bookworm main non-free contrib' > /etc/apt/sources.list
RUN apt update && \
    apt install -y --no-install-recommends ca-certificates wine64 \
    && apt clean && update-ca-certificates

RUN mkdir -p /home/server && useradd --uid 1000 --home /home/server --shell /bin/bash server \
    && chown -R 1000:1000 /home/server \
    && mkdir -p /app/bin/logs && chmod 0777 /app/bin/logs \
    && mkdir -p /app/bin/certs && chmod 0777 /app/bin/certs

ADD docker/certs/ /app/bin/certs/
RUN chmod -R 0777 /app/bin/certs

ADD dll/txmlconnector64-6.19.2.21.21.dll /app/bin/

EXPOSE 50051


FROM base as prod
ADD docker/app/entrypoint.sh /app/
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
