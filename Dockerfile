# Stage 1: Build stage
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files (if they exist)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED=0 is important for Alpine compatibility
# GOOS=linux ensures Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

# Stage 2: Runtime stage
FROM alpine:latest

# Install ca-certificates if your app needs HTTPS (optional)
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/purch .

# Expose port (adjust as needed for your app)
EXPOSE 8080

# Run the application
CMD ["./purch"]