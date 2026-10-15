FROM python:3.11-slim

ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG NO_PROXY=
ARG no_proxy=

ENV HTTP_PROXY= \
    HTTPS_PROXY= \
    http_proxy= \
    https_proxy= \
    NO_PROXY= \
    no_proxy= \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

# 安装系统依赖（PIL 和 pyzbar 需要）
# 用 Debian 官方源，不改写 /etc/apt/sources.list.d/*.sources。
# 之前改写到 mirrors.aliyun.com，在 python:3.11-slim(现为 Debian 13 trixie)
# 上偶发 `Err:4 ... trixie/main arm64 Packages` 导致构建中断；官方源更稳。
# 若国内拉取太慢，可自行换回镜像站（清华/中科大对 trixie 覆盖更完整）。
RUN printf 'Acquire::http::Proxy "false";\nAcquire::https::Proxy "false";\nAcquire::Retries "5";\n' > /etc/apt/apt.conf.d/99local-no-proxy && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      libgl1 \
      libglib2.0-0 \
      libzbar0 && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir \
    --index-url https://mirrors.aliyun.com/pypi/simple/ \
    --trusted-host mirrors.aliyun.com \
    -r requirements.txt

COPY main.py .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--log-level", "warning"]
