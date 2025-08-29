################################################
# Stage 1: Build PocketBase
################################################
FROM golang:1.24.3-bullseye AS backend-builder

# Set the working directory inside the container
WORKDIR /app

# Copy Go modules and download dependencies from pb/ directory
COPY pb/go.mod pb/go.sum ./
RUN go mod download

# Copy the source code from pb/ directory
COPY pb/ .

# Build the PocketBase application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o pocketbase main.go

################################################
# Stage 2: Create a smaller runtime image
################################################
FROM alpine:3.20

# Install necessary packages including wget for health checks
RUN apk --no-cache add ca-certificates libc6-compat wget

# Set working directory
WORKDIR /app

# Copy the compiled binary from the backend builder
COPY --from=backend-builder /app/pocketbase ./pocketbase

# Copy the schema file for database initialization
COPY --from=backend-builder /app/pb_bootstrap ./pb_bootstrap

# No frontend files needed - PocketBase is backend-only

# Create data directory for PocketBase
RUN mkdir -p /app/pb_data

# Expose PocketBase port
EXPOSE 8090

# Add health check for Kamal deployment
# Optimized timing for PocketBase initialization with schema loading
HEALTHCHECK --interval=5s --timeout=5s --start-period=45s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

# Run PocketBase server with data directory
ENTRYPOINT ["./pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/app/pb_data"]