ARG HOTSPOT_NODE_BASE_IMAGE=node:alpine
FROM ${HOTSPOT_NODE_BASE_IMAGE}
WORKDIR /app
ENV NPM_CONFIG_REGISTRY=https://registry.npmmirror.com
ENV npm_config_registry=https://registry.npmmirror.com
RUN npm install -g pnpm@10.28.0
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod
COPY build ./build
EXPOSE 3000
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0
CMD ["node", "build"]
