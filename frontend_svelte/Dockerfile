# ===========================================
# HotSpot Frontend (SvelteKit) Dockerfile
# Multi-stage build for production deployment
# ===========================================

# Stage 1: Build
FROM node:20-alpine AS builder
WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy dependency files first (for better caching)
COPY package.json pnpm-lock.yaml ./

# Install all dependencies (including devDependencies for build)
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build the SvelteKit app (adapter-node output to ./build)
RUN pnpm build

# Stage 2: Runtime
FROM node:20-alpine AS runtime
WORKDIR /app

# Install pnpm for production dependency install
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy package files
COPY package.json pnpm-lock.yaml ./

# Install production dependencies only
RUN pnpm install --frozen-lockfile --prod

# Copy built app from builder stage
COPY --from=builder /app/build ./build

# Expose port
EXPOSE 3000

# Environment variables
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0

# Start the Node.js server
CMD ["node", "build"]
