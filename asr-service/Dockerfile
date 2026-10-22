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
# 只作为模板放进镜像，不安装成生效的 config.yaml：
# config_loader 的取值顺序是「config.yaml(若存在) → 环境变量 → 默认」，
# 而示例文件里的 6 个字段值与 compose 传的 ASR_* 默认值完全相同，
# 所以装成 config.yaml 等于把 compose 的 ASR_* 全部变成死变量
# （想把模型换成 large 改 ASR_MODEL 无效）。
# 不用配置文件时走 ASR_* 环境变量；要用文件请自行挂载 config.yaml 到 /app/config.yaml。
COPY config.example.yaml ./config.example.yaml

COPY main.py .

EXPOSE 9000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "9000", "--log-level", "warning"]
