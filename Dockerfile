FROM mcr.microsoft.com/playwright:v1.48.2-jammy

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app

# 拷贝宿主机下载的 release 产物并解压
COPY release/xiaohongshu-mcp-linux-amd64.tar.gz /tmp/xhs.tar.gz
RUN tar -xzf /tmp/xhs.tar.gz -C /tmp && \
    mv /tmp/xiaohongshu-mcp-linux-amd64 /app/app && \
    chmod +x /app/app && \
    rm -f /tmp/xhs.tar.gz

# 创建共享目录并设置权限
RUN mkdir -p /app/images && \
    chmod 777 /app/images

# Playwright 镜像已包含 Chrome，无需额外设置

EXPOSE 18060

CMD ["./app"]
