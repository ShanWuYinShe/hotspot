FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app
COPY --from=builder /src/server .

EXPOSE 8081

ENV APP_PORT=8081
ENV APP_HOST=0.0.0.0

CMD ["./server"]
