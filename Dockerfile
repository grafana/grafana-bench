# syntax=docker/dockerfile:1.4.2-labs

FROM golang:1.23-alpine AS builder

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

# Install fixuid to allow setting the right uid when running local tests
RUN go get github.com/boxboat/fixuid@${FIXUID_VERSION} && \
  CGO_ENABLED=0 go build -o build/fixuid github.com/boxboat/fixuid

FROM grafana/k6:latest AS k6
FROM alpine:3.20 AS runtime

USER root
RUN apk add --no-cache ca-certificates git wget

## add bench user and group with known group id and user id
RUN addgroup -g 127 bench && \
  adduser --disabled-password -u 1001 -G bench bench

# Copy binaries
COPY --from=k6 /usr/bin/k6 /usr/local/bin/k6
COPY --from=builder /app/build/grafana-bench /usr/local/bin/grafana-bench
COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh

# Configure fixuid for dev builds only
ARG ENABLE_FIXUID=false
COPY --from=builder /app/build/fixuid /tmp/fixuid
RUN if [ "$ENABLE_FIXUID" = "true" ]; then \
      mkdir -p /etc/fixuid && \
      printf "user: bench\ngroup: bench\npaths:\n  - /home/bench\n  - /home/bench/tests\n  - /tmp\n" > /etc/fixuid/config.yml && \
      mv /tmp/fixuid /usr/local/bin/fixuid && \
      chown root:root /usr/local/bin/fixuid && \
      chmod 4755 /usr/local/bin/fixuid; \
    else \
      rm /tmp/fixuid; \
    fi

WORKDIR /home/bench

## ensure permissions on default test directory
RUN mkdir /home/bench/tests && chown -R bench:bench /home/bench/tests

USER bench

# call fixuid before calling bench to set the file permissions
# when running locally
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
