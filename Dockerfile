# 构建阶段：使用Go官方镜像编译程序
FROM golang:1.26-alpine3.23 AS builder

WORKDIR /build

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -installsuffix cgo -o app ./cmd

FROM alpine

RUN apk add --no-cache tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /build/app .
COPY etc/config-prod.yaml ./etc/config-prod.yaml

EXPOSE 8080     

CMD ["./app"]