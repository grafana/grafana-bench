#! /usr/bin/env sh
docker build --platform=linux/amd64 -t grafana-bench:latest .
docker tag grafana-bench:latest us.gcr.io/kubernetes-dev/hackathon-hg-bench:latest
docker push us.gcr.io/kubernetes-dev/hackathon-hg-bench:latest
