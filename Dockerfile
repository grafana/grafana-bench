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
  wget
  #chromium chromium-sandbox \
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
#libgstreamer-1.0.so.0 \
#libgtk-4.so.1 \
#libgraphene-1.0.so.0 \
#libatomic.so.1 \
#libwoff2dec.so.1.0.2 \
#libvpx.so.7 \
#libevent-2.1.so.7 \
#libgstallocators-1.0.so.0 \
#libgstapp-1.0.so.0 \
#libgstbase-1.0.so.0 \
#libgstpbutils-1.0.so.0 \
#libgstaudio-1.0.so.0 \
#libgstgl-1.0.so.0 \
#libgsttag-1.0.so.0 \
#libgstvideo-1.0.so.0
#libgstcodecparsers-1.0.so.0 \
#libgstfft-1.0.so.0 \
#libflite.so.1 \
#libflite_usenglish.so.1 \
#libflite_cmu_grapheme_lang.so.1 \
#libflite_cmu_grapheme_lex.so.1 \
#libflite_cmu_indic_lang.so.1 \
#libflite_cmu_indic_lex.so.1 \
#libflite_cmulex.so.1 \
#libflite_cmu_time_awb.so.1 \
#libflite_cmu_us_awb.so.1 \
#libflite_cmu_us_kal16.so.1 \
#libflite_cmu_us_kal.so.1 \
#libflite_cmu_us_rms.so.1 \
#libflite_cmu_us_slt.so.1 \
#libwebpdemux.so.2 \
#libavif.so.15 \
#libharfbuzz-icu.so.0 \
#libwebpmux.so.3 \
#libenchant-2.so.2 \
#libsecret-1.so.0 \
#libhyphen.so.0 \
#libmanette-0.2.so.0 \
#libGLESv2.so.2 \
#libx264.so


RUN apt install -y --no-install-recommends libasound2 libatk-bridge2.0-0 libatk1.0-0 libatspi2.0-0 libcairo2 libcups2 libdbus-1-3 libdrm2 libgbm1 libglib2.0-0 libnspr4 libnss3 libpango-1.0-0 libx11-6 libxcb1 libxcomposite1 libxdamage1 libxext6 libxfixes3 libxkbcommon0 libxrandr2 libcairo-gobject2 libdbus-glib-1-2 libfontconfig1 libfreetype6 libgdk-pixbuf-2.0-0 libgtk-3-0 libharfbuzz0b libpangocairo-1.0-0 libx11-xcb1 libxcb-shm0 libxcursor1 libxi6 libxrender1 libxtst6 libsoup-3.0-0 gstreamer1.0-libav gstreamer1.0-plugins-bad gstreamer1.0-plugins-base gstreamer1.0-plugins-good libegl1 libenchant-2-2 libepoxy0 libevdev2 libgles2 libglx0 libgstreamer-gl1.0-0 libgstreamer-plugins-base1.0-0 libgstreamer1.0-0 libgtk-4-1 libgudev-1.0-0 libharfbuzz-icu0 libhyphen0 libicu72 libjpeg62-turbo liblcms2-2 libmanette-0.2-0 libnotify4 libopengl0 libopenjp2-7 libopus0 libpng16-16 libproxy1v5 libsecret-1-0 libwayland-client0 libwayland-egl1 libwayland-server0 libwebp7 libwebpdemux2 libwoff1 libxml2 libxslt1.1 libatomic1 libevent-2.1-7 libavif15 xvfb fonts-noto-color-emoji fonts-unifont xfonts-scalable fonts-liberation fonts-ipafont-gothic fonts-wqy-zenhei fonts-tlwg-loma-otf fonts-freefont-ttf

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

#RUN yarn global add playwright
#RUN yarn global playwright install --with-deps

RUN npx playwright install --with-deps

USER bench

# call fixuid before calling bench to set the file permissions
# when running locally
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
