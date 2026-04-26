FROM golang:1.24-alpine

WORKDIR /app

# Install dependencies for Air and Go builds
RUN apk add --no-cache git gcc musl-dev

# Install Air for hot-reloading
RUN go install github.com/air-verse/air@latest

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Expose the API port
EXPOSE 8001

# Command to run the app with Air
CMD ["air", "-c", ".air.toml"]
