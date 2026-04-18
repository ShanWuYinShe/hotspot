FROM oven/bun:1 AS builder
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/build .
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json .
ENV PORT=3000
ENV HOST=0.0.0.0
EXPOSE 3000
CMD ["node", "index.js"]
