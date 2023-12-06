# syntax=docker/dockerfile:1.4.2-labs

FROM golang:1.21-alpine AS builder

ARG BENCH_REVISION 
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN apk add --no-cache ca-certificates git

# build bench
WORKDIR /app

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
                go build -ldflags="-X main.benchRevision=${BENCH_REVISION}" -trimpath -o grafana-bench ./cmd

FROM grafana/k6:latest AS k6
FROM alpine:3.18 AS runtime

USER root
RUN apk add --no-cache ca-certificates git chromium-swiftshader

RUN adduser -D -u 1010 -g 1010 bench

USER bench

# copy binaries
COPY --from=k6 /usr/bin/k6 /usr/local/bin/k6
COPY --from=builder /app/grafana-bench /usr/local/bin/grafana-bench

# config k6 browser
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/
ENV K6_BROWSER_HEADLESS=true
# no-sandbox chrome arg is required to run chrome browser in
# alpine and avoids the usage of SYS_ADMIN Docker capability
ENV K6_BROWSER_ARGS=no-sandbox

WORKDIR /home/bench
RUN mkdir /home/bench/tests


ENTRYPOINT ["grafana-bench"]
