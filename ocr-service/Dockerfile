FROM docker.m.daocloud.io/library/python:3.11-slim

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
RUN sed -i 's|http://deb.debian.org|https://deb.debian.org|g; s|http://security.debian.org|https://security.debian.org|g' /etc/apt/sources.list.d/*.sources 2>/dev/null || true && \
    printf 'Acquire::http::Proxy "false";\nAcquire::https::Proxy "false";\nAcquire::Retries "5";\n' > /etc/apt/apt.conf.d/99local-no-proxy && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      libgl1 \
      libglib2.0-0 \
      libzbar0 && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
