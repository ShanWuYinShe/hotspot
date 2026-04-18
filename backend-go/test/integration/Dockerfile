FROM golang:alpine

RUN apk add --no-cache curl

WORKDIR /test
COPY go.mod go.sum ./
COPY . .

RUN go mod download
