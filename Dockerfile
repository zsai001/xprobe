FROM golang:1.23.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

# 设置 MongoDB 连接 URI 环境变量
ENV MONGO_URI=mongodb://mongodb:27017

EXPOSE 8080

CMD ["./main"]