ARG GO_VERSION=1.26.2

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor ./vendor
RUN go list -mod=mod -m all >/dev/null

COPY . .
RUN CGO_ENABLED=0 go build -mod=mod -trimpath -ldflags="-s -w" -o /out/autodelete ./cmd/autodelete
RUN CGO_ENABLED=0 go build -mod=mod -trimpath -ldflags="-s -w" -o /out/autodelete-healthcheck ./cmd/healthcheck

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates tzdata \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd --system --gid 10001 autodelete \
  && useradd --system --uid 10001 --gid autodelete --home-dir /autodelete --shell /usr/sbin/nologin autodelete \
  && mkdir -p /autodelete/data \
  && chown -R autodelete:autodelete /autodelete

COPY --from=build /out/autodelete /usr/local/bin/autodelete
COPY --from=build /out/autodelete-healthcheck /usr/local/bin/autodelete-healthcheck

WORKDIR /autodelete
USER autodelete:autodelete

EXPOSE 2202
VOLUME ["/autodelete/data"]
STOPSIGNAL SIGTERM
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/usr/local/bin/autodelete-healthcheck"]

ENTRYPOINT ["/usr/local/bin/autodelete"]
