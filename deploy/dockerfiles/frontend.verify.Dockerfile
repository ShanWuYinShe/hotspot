ARG HOTSPOT_NODE_BASE_IMAGE=node:22.22.1-alpine3.23
FROM ${HOTSPOT_NODE_BASE_IMAGE}
WORKDIR /workspace
ENV NPM_CONFIG_REGISTRY=https://registry.npmmirror.com
ENV npm_config_registry=https://registry.npmmirror.com
RUN npm install -g pnpm@10.28.0
ENV HOME=/tmp/hotspot-home
ENV XDG_CACHE_HOME=/tmp/hotspot-home/.cache
ENV PNPM_STORE_DIR=/tmp/hotspot-home/.local/share/pnpm/store
