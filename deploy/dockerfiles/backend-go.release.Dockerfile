FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY backend-go /usr/local/bin/backend-go
EXPOSE 8080
ENV APP_HOST=0.0.0.0
ENV APP_PORT=8080
CMD ["backend-go"]
