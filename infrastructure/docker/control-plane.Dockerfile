FROM --platform=linux/amd64 golang:1.25.13-bookworm@sha256:40dfc169bd5ad8a8617e49c8ead7fe16c6873e79d6937539e9c2e5947b7984ef AS build

WORKDIR /src
COPY . .
RUN go build -trimpath -ldflags='-s -w' -o /out/haedes-control-plane ./services/control-plane/cmd/server

FROM --platform=linux/amd64 debian:bookworm-slim@sha256:f3034a6ec3c1205360777c4aae76234998866ad18806ae62b63a3f84ccad782b

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
