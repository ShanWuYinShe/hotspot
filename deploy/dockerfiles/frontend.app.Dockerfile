ARG HOTSPOT_NODE_BASE_IMAGE=node:24.14.0-alpine3.23
FROM ${HOTSPOT_NODE_BASE_IMAGE}
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
COPY node_modules ./node_modules
COPY build ./build
EXPOSE 3000
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0
CMD ["node", "build"]
