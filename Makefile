SHELL := /bin/bash
GO ?= go
BINDIR ?= bin
PKG := ./...

.DEFAULT_GOAL := build

build:
	$(GO) build -o $(BINDIR)/riskr ./cmd

clean:
	rm -rf $(BINDIR)

run-gateway: build
	./$(BINDIR)/riskr run gateway -c ./configs/config.example.yaml

run-streamer: build
	./$(BINDIR)/riskr run streamer -c ./configs/config.example.yaml

sim: build
	./$(BINDIR)/riskr sim -c ./configs/config.example.yaml clean

policy-apply: build
	./$(BINDIR)/riskr policy apply -c ./configs/config.example.yaml -f ./configs/policy.example.yaml

fmt:
	$(GO) fmt $(PKG)

lint:
	golangci-lint run

.PHONY: build clean run-gateway run-streamer sim policy-apply fmt lint
