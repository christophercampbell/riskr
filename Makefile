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
	./$(BINDIR)/riskr -c ./configs/config.example.yaml gateway

run-streamer: build
	./$(BINDIR)/riskr -c ./configs/config.example.yaml streamer

sim: build
	./$(BINDIR)/riskr -c ./configs/config.example.yaml sim -s clean

policy-apply: build
	./$(BINDIR)/riskr policy -c ./configs/config.example.yaml apply -f ./configs/policy.example.yaml

fmt:
	$(GO) fmt $(PKG)

lint:
	golangci-lint run

.PHONY: build clean run-gateway run-streamer sim policy-apply fmt lint
