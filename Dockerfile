# Stage 1: Build SvelteKit frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# Copy package files
COPY sk/package*.json ./
COPY sk/pnpm-lock.yaml ./

# Install dependencies 
RUN npm install -g pnpm && pnpm install

# Copy source code
COPY sk/ .

# Build the frontend
RUN pnpm build

################################################
# Stage 2: Build PocketBase
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
# Stage 3: Create a smaller runtime image
################################################
FROM alpine:3.20

# Install necessary packages
RUN apk --no-cache add ca-certificates libc6-compat

# Set working directory
WORKDIR /app

# Copy the compiled binary from the backend builder
COPY --from=backend-builder /app/pocketbase ./pocketbase

# Copy frontend build (SvelteKit static files) from frontend builder
COPY --from=frontend-builder /frontend/build ./pb_public

# Create data directory for PocketBase
RUN mkdir -p /app/pb_data

# Expose PocketBase port
EXPOSE 8090

# Run PocketBase server with data directory
ENTRYPOINT ["./pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/app/pb_data"]