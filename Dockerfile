# syntax=docker/dockerfile:1.23.0-labs@sha256:7eca9451d94f9b8ad22e44988b92d595d3e4d65163794237949a8c3413fbed5d

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG BENCH_REVISION
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
ARG FIXUID_VERSION=v0.6.0

RUN apk add --no-cache ca-certificates git

# build bench
WORKDIR /app

# go mod download first to cache modules for faster local builds
# TODO: check if we need to add the cache mounts to the go build steps bellow
COPY go.mod go.sum ./
RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
  --mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
  CGO_ENABLED=0 \
  go mod download -x

# now copy the rest of the source and build
COPY bench.go ./bench.go
COPY cmd ./cmd
COPY pkg ./pkg

RUN CGO_ENABLED=0 go build \
  -ldflags="-X github.com/grafana/grafana-bench/pkg/revision.bench=${BENCH_REVISION}" \
  -trimpath -o build/grafana-bench .

# Production slim image - no fixuid needed

FROM grafana/k6:2.0.0 AS k6
FROM alpine:3.23 AS runtime

USER root
RUN apk add --no-cache ca-certificates git wget

## add bench user and group with known group id and user id
RUN addgroup -g 127 bench && \
  adduser --disabled-password -u 1001 -G bench bench

# Copy binaries
COPY --from=k6 /usr/bin/k6 /usr/local/bin/k6
COPY --from=builder /app/build/grafana-bench /usr/local/bin/grafana-bench
COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh

# Create /tests directory as default test location
RUN mkdir -p /tests && chown -R bench:bench /tests

WORKDIR /tests

USER bench

# call fixuid before calling bench to set the file permissions
# when running locally
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
