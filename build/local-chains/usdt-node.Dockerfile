FROM ghcr.io/foundry-rs/foundry:latest

USER root

RUN set -eux; \
    if command -v apt-get >/dev/null 2>&1; then \
      apt-get update; \
      apt-get install -y --no-install-recommends jq ca-certificates; \
      rm -rf /var/lib/apt/lists/*; \
    elif command -v apk >/dev/null 2>&1; then \
      apk add --no-cache jq ca-certificates; \
    else \
      echo "unsupported base image package manager"; \
      exit 1; \
    fi
