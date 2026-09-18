# syntax=docker/dockerfile:1.7

FROM --platform=linux/amd64 rust:1.80.1-bookworm@sha256:b6f344496c19f4a91adffeb1c6f3975ed4f0c134965451cc96505002f8a7bd1a AS builder

WORKDIR /src
COPY Cargo.toml Cargo.lock ./
COPY services/sandbox-runtime/Cargo.toml services/sandbox-runtime/Cargo.toml
COPY services/sandbox-runtime services/sandbox-runtime
RUN cargo build --locked --release -p haedes-sandbox-runtime

FROM --platform=linux/amd64 debian:bookworm-20251103-slim@sha256:f522a3167fb670cc4a4518fab6d8c4227fe4bc458f4c7294832383bf284d5e78

ARG RUNTIME_VERSION=0.1.0

LABEL org.opencontainers.image.title="haedes sandbox base" \
      org.opencontainers.image.description="Non-root Linux development sandbox with the haedes runtime" \
      org.opencontainers.image.version="${RUNTIME_VERSION}"

ENV DEBIAN_FRONTEND=noninteractive \
    HOME=/home/sandbox \
    TMPDIR=/tmp \
    HAEDES_RUNTIME_BIND=0.0.0.0:8080 \
    HAEDES_RUNTIME_WORKSPACE=/workspace

RUN apt-get update \
    && apt-get install --no-install-recommends --yes \
        bash \
        ca-certificates \
        cargo \
        curl \
        gcc \
        g++ \
        git \
        golang \
        make \
        nodejs \
        npm \
        python3 \
        python3-pip \
        rustc \
        wget \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --home-dir /home/sandbox --shell /usr/sbin/nologin --uid 10001 sandbox \
    && mkdir -p /workspace /tmp \
    && chown -R sandbox:sandbox /home/sandbox /workspace /tmp \
    && chmod 1777 /tmp

COPY --from=builder /src/target/release/haedes-sandbox-runtime /usr/local/bin/haedes-runtime
COPY images/sandbox-base/entrypoint.sh /usr/local/bin/haedes-entrypoint
RUN chmod 0755 /usr/local/bin/haedes-entrypoint /usr/local/bin/haedes-runtime

EXPOSE 8080
USER sandbox
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/haedes-entrypoint"]
