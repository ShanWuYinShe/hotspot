FROM golang:1.26-alpine

RUN apk add --no-cache curl

WORKDIR /test
COPY go.mod go.sum ./
COPY . .

RUN go mod download
