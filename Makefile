CGO_ENABLED=1
export CGO_ENABLED
NAME=cni-ipvlan-vpc-k8s
VERSION:=$(shell git describe --tags)
DOCKER_IMAGE=lyft/cni-ipvlan-vpc-k8s:$(VERSION)
export GO111MODULE=on

.PHONY: all
all: build test

.PHONY: clean
clean:
	rm -f *.tar.gz $(NAME)-*

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
ifndef GOOS
	go test -v ./aws/... ./nl ./cmd/cni-ipvlan-vpc-k8s-tool ./lib/...
else
	@echo Tests not available when cross-compiling
endif

.PHONY: build
build:
	go build -i -o $(NAME)-ipam ./plugin/ipam/main.go
	go build -i -o $(NAME)-ipvlan ./plugin/ipvlan/ipvlan.go
	go build -i -o $(NAME)-unnumbered-ptp ./plugin/unnumbered-ptp/unnumbered-ptp.go
	go build -i -ldflags "-X main.version=$(VERSION)" -o $(NAME)-tool ./cmd/cni-ipvlan-vpc-k8s-tool/cni-ipvlan-vpc-k8s-tool.go

	tar cvzf cni-ipvlan-vpc-k8s-${GOARCH}-$(VERSION).tar.gz $(NAME)-ipam $(NAME)-ipvlan $(NAME)-unnumbered-ptp $(NAME)-tool

.PHONY: test-docker
test-docker:
	docker build -t $(DOCKER_IMAGE) .

.PHONY: build-docker
build-docker: test-docker
	docker run --rm -v $(PWD):/dist:rw $(DOCKER_IMAGE) bash -exc 'cp /go/src/github.com/lyft/cni-ipvlan-vpc-k8s/cni-ipvlan-vpc-k8s-$(VERSION).tar.gz /dist'

.PHONY: interactive-docker
interactive-docker: test-docker
	docker run --privileged -v $(PWD):/go/src/github.com/lyft/cni-ipvlan-vpc-k8s -it $(DOCKER_IMAGE) /bin/bash

.PHONY: ci
ci:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.30.0
	$(MAKE) all
	$(MAKE) lint
