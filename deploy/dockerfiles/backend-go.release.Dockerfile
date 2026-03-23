ARG HOTSPOT_ALPINE_BASE_IMAGE=alpine:latest
FROM ${HOTSPOT_ALPINE_BASE_IMAGE}
WORKDIR /app
ARG HOTSPOT_ALPINE_REPO=https://mirrors.ustc.edu.cn/alpine
RUN sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${HOTSPOT_ALPINE_REPO}#g" /etc/apk/repositories && apk add --no-cache ca-certificates curl
COPY backend-go /usr/local/bin/backend-go
EXPOSE 8080
ENV APP_HOST=0.0.0.0
ENV APP_PORT=8080
CMD ["backend-go"]
