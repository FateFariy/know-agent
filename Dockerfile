# 构建阶段：使用Go官方镜像编译程序
FROM dhi.io/golang:1.26-alpine3.23 AS builder

WORKDIR /build

# 优先复制依赖文件，利用Docker缓存
ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并构建静态二进制（无系统依赖）
COPY . .
RUN go build -ldflags="-s -w" -installsuffix cgo -o app ./main.go

# 运行阶段：轻量Alpine镜像
FROM alpine

# 配置时区（可选）
RUN apk add --no-cache tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone
ENV TZ=Asia/Shanghai

WORKDIR /app

# 复制构建产物
COPY --from=builder /build/app .

# 暴露端口并启动
EXPOSE 8080
CMD ["./app"]