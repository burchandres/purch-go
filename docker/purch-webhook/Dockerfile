# Stage 1: Build stage
FROM golang:1.26-alpine AS dependencies

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED=0 is important for Alpine compatibility
# GOOS=linux ensures Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o purch-webhook ./webhook/cmd/main.go

# Stage 2: Runtime stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=dependencies --chmod=755 /app/purch-webhook .

# Expose port
EXPOSE 8081

# Run the application
CMD ["./purch-webhook"]