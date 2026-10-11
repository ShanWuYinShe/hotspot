ARG GO_VERSION=1.26.9
ARG ALPINE_VERSION=3.24.1

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# dl-cdn.alpinelinux.org（Fastly CDN）在国内网络劣化严重（~30KB/s），
# 换 aliyun 镜像源，避免 apk 下载索引/包超时
RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.aliyun.com#g' /etc/apk/repositories && apk add --no-cache git

WORKDIR /src

ARG GOPROXY
# 默认走 goproxy.cn（proxy.golang.org 在国内不可达），compose 亦可显式传参覆盖
ENV GOPROXY=${GOPROXY:-https://goproxy.cn,direct}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 版本指纹（由部署侧传入：scripts/deploy-production.sh 自动取 git describe /
# 完整 sha / UTC 时间；本地直接 build 不传时兜底 dev，不破坏层缓存）。
ARG GIT_VERSION=dev
ARG GIT_COMMIT=
ARG GIT_BUILD_TIME=

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w \
      -X hotspot/backend-go/internal/platform/buildinfo.Version=${GIT_VERSION} \
      -X hotspot/backend-go/internal/platform/buildinfo.Commit=${GIT_COMMIT} \
      -X hotspot/backend-go/internal/platform/buildinfo.BuildTime=${GIT_BUILD_TIME}" \
    -o server ./cmd/server

FROM alpine:${ALPINE_VERSION}

# ARG 按 stage 作用域：第二阶段重新声明才能在 LABEL 中引用全局 build args
ARG GIT_VERSION=dev
ARG GIT_COMMIT=
ARG GIT_BUILD_TIME=

# OCI 标准镜像标签：docker inspect 即可核对镜像对应的源码版本
LABEL org.opencontainers.image.version=${GIT_VERSION} \
      org.opencontainers.image.revision=${GIT_COMMIT} \
      org.opencontainers.image.created=${GIT_BUILD_TIME}

RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.aliyun.com#g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata curl

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /src/server .

EXPOSE 8081

ENV APP_PORT=8081
ENV APP_HOST=0.0.0.0

USER app
CMD ["./server"]
