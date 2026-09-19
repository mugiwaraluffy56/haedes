FROM golang:1.24-bookworm AS build

WORKDIR /src
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/haedes-control-plane ./services/control-plane/cmd/server

FROM debian:bookworm-slim

ARG BUILD_SHA=dev
LABEL org.opencontainers.image.title="haedes control plane" \
      org.opencontainers.image.revision="${BUILD_SHA}"

RUN apt-get update \
    && apt-get install --no-install-recommends --yes ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 10001 --shell /usr/sbin/nologin haedes

COPY --from=build /out/haedes-control-plane /usr/local/bin/haedes-control-plane

USER haedes
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/haedes-control-plane"]
