FROM node:20-alpine
WORKDIR /workspace
RUN corepack enable && corepack prepare pnpm@10.28.0 --activate
ENV PNPM_STORE_DIR=/cache/pnpm/store
