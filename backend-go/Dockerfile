FROM --platform=$BUILDPLATFORM golang:1.26.2-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

ARG GOPROXY
ENV GOPROXY=${GOPROXY:-direct}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -ldflags="-s -w" -o server ./cmd/server

FROM alpine:3.22.3

RUN apk add --no-cache ca-certificates tzdata curl

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /src/server .

EXPOSE 8081

ENV APP_PORT=8081
ENV APP_HOST=0.0.0.0

USER app
CMD ["./server"]
