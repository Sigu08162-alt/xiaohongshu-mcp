FROM debian:bookworm-slim

ENV TZ=Asia/Shanghai \
    PLAYWRIGHT_BROWSERS_PATH=/root/.cache/ms-playwright

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone && \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates fonts-liberation \
    libasound2 libatk-bridge2.0-0 libatk1.0-0 libcairo2 libcups2 \
    libdbus-1-3 libexpat1 libfontconfig1 libgbm1 libglib2.0-0 \
    libgtk-3-0 libnspr4 libnss3 libpango-1.0-0 libpangocairo-1.0-0 \
    libx11-6 libx11-xcb1 libxcb1 libxcomposite1 libxcursor1 \
    libxdamage1 libxext6 libxfixes3 libxi6 libxrandr2 libxrender1 \
    libxss1 libxtst6 xdg-utils \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app

COPY release/xiaohongshu-mcp-linux-amd64 /app/app
RUN chmod +x /app/app

COPY configs/ /app/configs/

RUN mkdir -p /app/images /app/data && \
    chmod 777 /app/images /app/data

EXPOSE 18060

CMD ["./app"]
