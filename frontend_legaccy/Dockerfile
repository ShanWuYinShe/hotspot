# Stage 1: Build
FROM node:22-alpine AS builder

# Enable pnpm via corepack
RUN corepack enable

WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build application
RUN pnpm run build

# Stage 2: Runtime
FROM node:22-alpine AS runner

WORKDIR /app

# Set environment variables
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000

# Copy built artifacts from builder
# Nuxt 3/4 builds to .output directory by default
COPY --from=builder /app/.output ./.output

# Expose port
EXPOSE 3000

# Run the server
CMD ["node", ".output/server/index.mjs"]
