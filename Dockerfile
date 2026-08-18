# ------------------------------------------------------------------------------
# Stage 1: Build Application Binary with Layered Dependency Caching
# ------------------------------------------------------------------------------
FROM rust:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache musl-dev openssl-dev pkgconfig

# Copy dependency manifests first to cache compiled crates
COPY Cargo.toml Cargo.lock* ./

# Create dummy entrypoint to pre-build dependencies into Docker layer cache
RUN mkdir src && echo "fn main() {}" > src/main.rs \
    && cargo build --release \
    && rm -rf src

# Copy actual source code and build final binary
COPY src ./src
RUN touch src/main.rs && cargo build --release

# ------------------------------------------------------------------------------
# Stage 2: Minimal Production Runtime Container (~25 MB total)
# ------------------------------------------------------------------------------
FROM alpine:3.20

WORKDIR /app

# Install CA certificates for TLS
RUN apk add --no-cache ca-certificates

# Copy compiled binary from builder stage
COPY --from=builder /app/target/release/moarchan /app/moarchan

# Copy static assets (CSS, JS, images, templates)
COPY static ./static

# Ensure upload directory exists
RUN mkdir -p ./static/images/uploads

EXPOSE 9001

CMD ["/app/moarchan"]
