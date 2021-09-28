FROM golang:1.10.3 AS builder
LABEL maintainer="mcutalo@lyft.com"

WORKDIR /go/src/github.com/lyft/cni-ipvlan-vpc-k8s/

RUN go get github.com/golang/dep && \
  go install github.com/golang/dep/cmd/dep && \
  go get -u gopkg.in/alecthomas/gometalinter.v2 && \
  gometalinter.v2 --install

COPY . /go/src/github.com/lyft/cni-ipvlan-vpc-k8s/

RUN dep ensure -v && make build
