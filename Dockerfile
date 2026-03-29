# 构建阶段
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /db-backup ./cmd/server

# 运行阶段
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata mysql-client postgresql-client

COPY --from=builder /db-backup /usr/local/bin/db-backup

RUN mkdir -p /var/lib/db-backup /etc/db-backup

EXPOSE 8080

ENTRYPOINT ["db-backup"]
CMD ["-config", "/etc/db-backup/config.yaml"]
