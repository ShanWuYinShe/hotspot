# Stage 1: Plan
FROM lukemathwalker/cargo-chef:latest-rust-1.81-alpine AS chef
WORKDIR /app

# Stage 2: Prepare recipe
FROM chef AS planner
COPY . .
RUN cargo chef prepare --recipe-path recipe.json

# Stage 3: Build dependencies
FROM chef AS builder
COPY --from=planner /app/recipe.json recipe.json
# Build dependencies - this is the caching layer!
RUN cargo chef cook --release --recipe-path recipe.json

# Stage 4: Build application
COPY . .
# SQLx requires DATABASE_URL to check queries at compile time unless offline mode is used
# We assume offline mode (sqlx-data.json) is present or we skip query checking.
# For simplicity in this template, we set SQLX_OFFLINE=true if data is present, 
# otherwise one might need a running DB during build. 
# Here we assume sqlx-data.json is up to date or we skip validation.
ENV SQLX_OFFLINE=true
RUN cargo build --release --bin hotspot_backend

# Stage 5: Runtime
FROM alpine:3.19 AS runtime
WORKDIR /app

# Install necessary runtime dependencies
# libpq is required for postgres, ca-certificates for HTTPS
RUN apk add --no-cache libpq ca-certificates tzdata

# Copy the compiled binary
COPY --from=builder /app/target/release/hotspot_backend /app/hotspot_backend

# Copy configuration/migration files if needed at runtime
# (Migrations are usually embedded or run via cli, assuming embedded or external management)
# If using sqlx migrate at runtime in code, no extra files needed if embedded.
# If config is via env, no config file needed.

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/hotspot_backend"]
