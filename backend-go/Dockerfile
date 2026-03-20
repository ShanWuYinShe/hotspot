FROM golang:1.26-alpine AS base
WORKDIR /app
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download

FROM base AS source
COPY . .

FROM source AS test
CMD ["go", "test", "./..."]

FROM source AS build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/backend-go ./cmd/server

FROM alpine:3.21 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /out/backend-go /usr/local/bin/backend-go
EXPOSE 8080
ENV APP_HOST=0.0.0.0
ENV APP_PORT=8080
CMD ["backend-go"]
