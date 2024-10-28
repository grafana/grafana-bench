# syntax=docker/dockerfile:1.4.2-labs

FROM golang:1.22-alpine AS builder

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

RUN go get github.com/boxboat/fixuid@${FIXUID_VERSION} && \
    CGO_ENABLED=0 go build -o build/fixuid github.com/boxboat/fixuid

FROM grafana/k6:latest AS k6
FROM debian:12.7-slim AS runtime

USER root
RUN apt update && apt install --no-install-recommends -y \
    ca-certificates \
    git \
    wget \
    chromium chromium-sandbox \
    npm

RUN npm install -g yarn

RUN addgroup --gid 127 bench && \
    adduser --disabled-password --uid 1001 --gid 127 bench && \
    apt clean

RUN mkdir -p /etc/fixuid && \
    printf "user: bench\ngroup: bench\n" > /etc/fixuid/config.yml

# copy binaries
COPY --from=k6 /usr/bin/k6 /usr/local/bin/k6
COPY --from=builder /app/build/grafana-bench /usr/local/bin/grafana-bench
COPY --from=builder /app/build/fixuid /usr/local/bin/
COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chown root:root /usr/local/bin/fixuid && \
    chmod 4755 /usr/local/bin/fixuid

# config playwright
ENV PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium

# config k6 browser
ENV CHROME_BIN=/usr/bin/chromium
ENV CHROME_PATH=/usr/lib/chromium/
ENV K6_BROWSER_HEADLESS=true
# no-sandbox chrome arg is required to run chrome browser in
# alpine and avoids the usage of SYS_ADMIN Docker capability
ENV K6_BROWSER_ARGS=no-sandbox

WORKDIR /home/bench

RUN mkdir /home/bench/tests

RUN chown -R bench:bench /home/bench/tests

USER bench

RUN wget -qO- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash 

RUN nvm install 20

# ENTRYPOINT ["grafana-bench"]
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
