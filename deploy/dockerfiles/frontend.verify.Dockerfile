FROM node:20-alpine
WORKDIR /workspace
RUN corepack enable && corepack prepare pnpm@10.28.0 --activate
ENV HOME=/tmp/hotspot-home
ENV XDG_CACHE_HOME=/tmp/hotspot-home/.cache
ENV PNPM_STORE_DIR=/cache/pnpm/store
