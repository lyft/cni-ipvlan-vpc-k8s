FROM golang:1.17-buster as builder

WORKDIR /go/src/github.com/lyft/cni-ipvlan-vpc-k8s/
COPY . /go/src/github.com/lyft/cni-ipvlan-vpc-k8s/

RUN make build
