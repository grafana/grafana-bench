FROM golang:1.20-alpine3.17 AS builder

ARG GRAFANA_TEST_REPO

RUN apk update && apk add --no-cache git

# install mage
RUN git clone https://github.com/magefile/mage
RUN cd mage && go run bootstrap.go

# build bench
WORKDIR /app
COPY go.mod go.mod
RUN go mod download

COPY bench bench/
COPY Magefile.go Magefile.go

RUN go mod tidy

RUN mage -compile ./grafana-bench

FROM grafana/k6:latest

## Run container
COPY --from=builder /app/grafana-bench /usr/local/bin/grafana-bench

USER root

## this is a hack. we shouldn't need go installed
RUN apk update && apk add --no-cache go

COPY docker_startup.sh docker_startup.sh
RUN chmod +x docker_startup.sh

RUN mkdir -p /home/k6/work/test

## this is also a hack. should figure out how to push tests to container better
COPY work/test/suite /home/k6/work/test/suite/

WORKDIR /home/k6

ENTRYPOINT ["./docker_startup.sh"]
