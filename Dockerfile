FROM alpine:3.15@sha256:19b4bcc4f60e99dd5ebdca0cbce22c503bbcff197549d7e19dab4f22254dc864
LABEL maintainer="jonsosnyan <https://jonssonyan.com>"
WORKDIR /tpdata/trojan-panel/
ENV mariadb_ip=127.0.0.1 \
    mariadb_port=9507 \
    mariadb_user=root \
    mariadb_pas=123456 \
    redis_host=127.0.0.1 \
    redis_port=6378 \
    redis_pass=123456 \
    server_port=8081 \
    TZ=Asia/Shanghai \
    GIN_MODE=release
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
COPY build/trojan-panel-${TARGETOS}-${TARGETARCH}${TARGETVARIANT} trojan-panel
# Set apk China mirror
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add bash tzdata ca-certificates && \
    rm -rf /var/cache/apk/*
ENTRYPOINT chmod 777 ./trojan-panel && \
    ./trojan-panel \
    -host=${mariadb_ip} \
    -port=${mariadb_port} \
    -user=${mariadb_user} \
    -password=${mariadb_pas} \
    -redisHost=${redis_host} \
    -redisPort=${redis_port} \
    -redisPassword=${redis_pass} \
    -serverPort=${server_port}
