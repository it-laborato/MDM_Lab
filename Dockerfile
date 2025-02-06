# Stage 1: Use Node.js base image for building assets
FROM node:20.18.1-alpine AS node-builder
RUN apk add --no-cache make git
WORKDIR /app
# RUN npm install -g yarn
COPY . . 

RUN make deps-js
RUN make generate-js 
# COPY . .


# Stage 2: Build the Go binary
FROM golang:1.20-alpine AS go-builder

WORKDIR /app

# Copy the project (including assets generated in the previous stage)
COPY --from=node-builder /app .

# Install Go dependencies and build the app
RUN make deps-go generate-go 
# RUN go build -o mdmlab .
RUN make build
# Stage 3: Create a lightweight final image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=go-builder /app/mdmlab .
# Copy any required assets (e.g., templates, static files)
COPY --from=go-builder /app/assets ./assets

# Expose the port your app listens on
EXPOSE 8080

# Command to run when the container starts
