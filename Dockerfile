# syntax=docker/dockerfile:1.4.2-labs
FROM grafana/k6:0.46.0 AS k6

FROM golang:1.21-alpine AS builder

RUN apk add --no-cache ca-certificates git

# build bench
WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH

# go mod download first to cache modules for faster local builds
COPY go.mod go.sum ./
RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
        --mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
            CGO_ENABLED=0 \
                go mod download -x

# now copy the rest of the source and build
COPY cmd ./cmd
COPY bench ./bench
RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
        --mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
            CGO_ENABLED=0 \
                go build -trimpath -o grafana-bench ./cmd

FROM alpine:3.18 AS runtime

RUN apk add --no-cache ca-certificates git
RUN adduser -D -u 1010 -g 1010 bench

COPY docker_startup.sh /home/bench/docker_startup.sh
COPY --from=k6 /usr/bin/k6 /usr/local/bin/k6
COPY --from=builder /app/grafana-bench /usr/local/bin/grafana-bench

USER bench
WORKDIR /home/bench

ENTRYPOINT [ "./docker_startup.sh" ]
