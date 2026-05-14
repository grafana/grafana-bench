# syntax=docker/dockerfile:1.23.0-labs@sha256:7eca9451d94f9b8ad22e44988b92d595d3e4d65163794237949a8c3413fbed5d

FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

ARG BENCH_REVISION 
ARG TARGETOS=linux
ARG TARGETARCH=amd64
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

FROM grafana/k6:latest@sha256:632ddbc81a4a9fdc9e597da91ab1d8fcf1916dd988b43b4a4559d2f8d8e73d47 AS k6
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS runtime

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
