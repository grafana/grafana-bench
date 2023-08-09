FROM golang:1.20-alpine3.17 AS builder

ARG GRAFANA_TEST_REPO

RUN apk update && apk add --no-cache git

# build bench
WORKDIR /app
COPY go.mod go.sum .
RUN go mod download

COPY bench bench/
COPY cmd cmd/

RUN go build -o grafana-bench ./cmd

FROM grafana/k6:latest
USER root

## Run container
COPY docker_startup.sh docker_startup.sh
RUN chmod +x docker_startup.sh

# get test suite # this is a hack. should get test suite via build arg
RUN mkdir -p /home/k6/work/test
COPY work/test/suite /home/k6/work/test/suite/

# copy binary
COPY --from=builder /app/grafana-bench /usr/local/bin/grafana-bench

WORKDIR /home/k6

ENTRYPOINT ["./docker_startup.sh"]
