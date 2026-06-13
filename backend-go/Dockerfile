ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.22

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

ARG GOPROXY
ENV GOPROXY=${GOPROXY:-direct}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -ldflags="-s -w" -o server ./cmd/server

FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache ca-certificates tzdata curl

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /src/server .

EXPOSE 8081

ENV APP_PORT=8081
ENV APP_HOST=0.0.0.0

USER app
CMD ["./server"]
