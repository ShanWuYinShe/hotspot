ARG GO_VERSION=1.26.5
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

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -ldflags="-s -w" -o server ./cmd/server

FROM alpine:${ALPINE_VERSION}

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
