FROM mcr.microsoft.com/playwright:v1.48.2-jammy

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 设置 Playwright 环境变量，使用镜像预装的浏览器
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

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

EXPOSE 18060

CMD ["./app"]
