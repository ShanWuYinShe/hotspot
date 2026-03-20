# ===========================================
# HotSpot Frontend (SvelteKit) Dockerfile
# Multi-stage build for production deployment
# ===========================================

FROM node:20-alpine AS base
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.28.0 --activate
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

FROM base AS source
ARG VITE_AMAP_KEY=
ARG VITE_AMAP_SERVICE_HOST=/_AMapService
ENV VITE_AMAP_KEY=$VITE_AMAP_KEY
ENV VITE_AMAP_SERVICE_HOST=$VITE_AMAP_SERVICE_HOST
COPY . .

FROM source AS validate
CMD ["sh", "-c", "pnpm check && pnpm build --mode test"]

FROM source AS builder
RUN pnpm build

FROM node:20-alpine AS runtime
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.28.0 --activate
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod
COPY --from=builder /app/build ./build
EXPOSE 3000
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0
CMD ["node", "build"]
