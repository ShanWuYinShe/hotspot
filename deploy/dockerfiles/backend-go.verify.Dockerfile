FROM golang:1.26-alpine
WORKDIR /workspace
RUN apk add --no-cache ca-certificates git bash
ENV HOME=/tmp/hotspot-home
ENV XDG_CACHE_HOME=/tmp/hotspot-home/.cache
ENV GOMODCACHE=/cache/go/pkg
ENV GOCACHE=/cache/go/build
