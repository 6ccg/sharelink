# Step 1: Build the React frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm config set registry https://registry.npmmirror.com
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Step 2: Build the Go backend
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY backend/ ./
RUN go build -o server ./cmd/server

# Step 3: Runner stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Copy binaries and assets
COPY --from=backend-builder /app/server .
COPY --from=frontend-builder /app/dist ./frontend/dist
COPY backend/data/ip2region.xdb /app/data/ip2region.xdb
RUN mkdir -p /data \
    && addgroup -S sharelink && adduser -S -G sharelink sharelink \
    && chown -R sharelink:sharelink /app /data

# Default environmental variables
ENV PORT=8080
ENV DB_TYPE=sqlite
ENV DB_DSN=/data/sharelink.db
ENV DATA_DIR=/data
ENV IP_DB_PATH=/app/data/ip2region.xdb
ENV LOG_LEVEL=info

EXPOSE 8080

USER sharelink
CMD ["./server"]
