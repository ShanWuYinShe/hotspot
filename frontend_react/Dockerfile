FROM oven/bun:1.3.14-alpine AS builder
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
ARG VITE_AMAP_KEY
ARG VITE_AMAP_SECURITY_CODE
ARG VITE_AMAP_SERVICE_HOST
ENV VITE_AMAP_KEY=$VITE_AMAP_KEY
ENV VITE_AMAP_SECURITY_CODE=$VITE_AMAP_SECURITY_CODE
ENV VITE_AMAP_SERVICE_HOST=$VITE_AMAP_SERVICE_HOST
COPY . .
RUN bun run build

FROM nginx:1.30.1-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
