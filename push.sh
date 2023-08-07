#! /usr/bin/env sh
docker build --platform=linux/amd64 --tag grafana-bench:latest .
docker tag grafana-bench:latest us.gcr.io/kubernetes-dev/hackathon-hg-bench:latest
docker push us.gcr.io/kubernetes-dev/hackathon-hg-bench:latest
