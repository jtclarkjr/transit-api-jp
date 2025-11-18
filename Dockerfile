# Use the official Golang image as the build stage
ARG GO_VERSION=1.25.1
FROM golang:${GO_VERSION}-bookworm AS builder

# Set the working directory
WORKDIR /usr/src/app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the source code and build the application
COPY . .
RUN go build -v -o /run-app .

# Use the official Debian image as the final stage
FROM debian:bookworm

# Install CA certificates
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

# Copy the built application from the builder stage
COPY --from=builder /run-app /usr/local/bin/

# Command to run the application
CMD ["run-app"]

