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
    PIP_NO_CACHE_DIR=1 \
    HF_HOME=/root/.cache/huggingface

WORKDIR /app

RUN sed -i 's|deb\.debian\.org|mirrors.aliyun.com|g; s|security\.debian\.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/*.sources 2>/dev/null || true && \
    printf 'Acquire::http::Proxy "false";\nAcquire::https::Proxy "false";\nAcquire::Retries "5";\n' > /etc/apt/apt.conf.d/99local-no-proxy && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      libgomp1 && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir \
    --index-url https://mirrors.aliyun.com/pypi/simple/ \
    --trusted-host mirrors.aliyun.com \
    -r requirements.txt

COPY config_loader.py .
COPY config.example.yaml ./config.yaml

COPY main.py .

EXPOSE 9000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "9000", "--log-level", "warning"]
