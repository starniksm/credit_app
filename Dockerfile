# Use the official Golang image to create a build artifact
FROM golang:1.21-alpine AS builder

# Set destination for COPY
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a minimal alpine image for the final image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy static files
COPY --from=builder /app/static ./static

# Copy fonts for PDF generation
COPY --from=builder /app/fonts ./fonts

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["./main"]