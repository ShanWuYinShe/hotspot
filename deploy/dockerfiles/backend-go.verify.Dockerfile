ARG HOTSPOT_GOLANG_BASE_IMAGE=golang:alpine
FROM ${HOTSPOT_GOLANG_BASE_IMAGE}
WORKDIR /workspace
ARG HOTSPOT_ALPINE_REPO=https://mirrors.ustc.edu.cn/alpine
RUN sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${HOTSPOT_ALPINE_REPO}#g" /etc/apk/repositories && apk add --no-cache ca-certificates git bash
ENV HOME=/tmp/hotspot-home
ENV XDG_CACHE_HOME=/tmp/hotspot-home/.cache
ENV GOPATH=/tmp/hotspot-go
ENV GOMODCACHE=/tmp/hotspot-go/pkg/mod
ENV GOCACHE=/tmp/hotspot-home/.cache/go-build
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
