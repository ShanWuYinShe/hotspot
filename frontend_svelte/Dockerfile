FROM oven/bun:1.3-alpine AS builder
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

FROM node:22-alpine3.22
WORKDIR /app
COPY --from=builder /app/build .
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json .
ENV PORT=3000
ENV HOST=0.0.0.0
EXPOSE 3000
CMD ["node", "index.js"]
