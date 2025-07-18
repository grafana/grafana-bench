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
FROM debian:12.11-slim AS runtime

USER root
RUN apt update && apt install --no-install-recommends -y \
  ca-certificates \
  git \
  wget \
  chromium chromium-sandbox
# browser deps- left here for debugging purposes.
# `playwright install --with-deps` requires sudo access
# however when we install the deps directly, without chrome
# we get test timeouts
#libglib2.0-0 \ 
#libnss3 \ 
#libnspr4 \
#libdbus-1-3 \
#libatk1.0-0 \
#libatspi2.0-0 \
#libx11-6 \
#libxcomposite1 \
#libxdamage1 \
#libxext6 \
#libxfixes3 \
#libxrandr2 \
#libgbm1 \
#libxcb1 \
#libxkbcommon0 \
#libasound2

RUN wget -qO- https://deb.nodesource.com/setup_20.x | bash

RUN apt install -y nodejs

RUN npm install -g yarn

RUN addgroup --gid 127 bench && \
  adduser --disabled-password --uid 1001 --gid 127 bench && \
  apt clean

# configure fixuid to map the bench user to the uid:gui of the user invoking the image
RUN mkdir -p /etc/fixuid && \
  printf "user: bench\ngroup: bench\n" > /etc/fixuid/config.yml

# copy binaries
COPY --from=k6 /usr/bin/k6 /usr/local/bin/k6
COPY --from=builder /app/build/grafana-bench /usr/local/bin/grafana-bench
COPY --from=builder /app/build/fixuid /usr/local/bin/
COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chown root:root /usr/local/bin/fixuid && \
  chmod 4755 /usr/local/bin/fixuid

WORKDIR /home/bench

RUN mkdir /home/bench/tests /home/bench/.cache && \
  chown -R bench:bench /home/bench/tests /home/bench/.cache

USER bench

# call fixuid before calling bench to set the file permissions
# when running locally
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
