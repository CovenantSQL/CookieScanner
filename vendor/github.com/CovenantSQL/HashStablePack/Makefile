
# NOTE: This Makefile is only necessary if you 
# plan on developing the hsp tool and library.
# Installation can still be performed with a
# normal `go install`.

# generated integration test files
GGEN = ./test/covenantsql.go

SHELL := /bin/bash

BIN = $(GOBIN)/hsp

.PHONY: clean wipe install get-deps bench all

$(BIN):
	cd hsp && go build . && go install .

install: $(BIN)

GGEN:
	go generate ./test/covenantsql.go


test: all
	go test -v ./...

bench: all
	go test -bench ./...

clean:
	$(RM) GGEN

wipe: clean
	$(RM) $(BIN)

get-deps:
	go get -d -t ./...

all: install GGEN

# travis CI enters here
travis:
	go get -d -t ./...
	cd hsp && go build -o "$${GOPATH%%:*}/bin/hsp" .
	go generate ./test
	go test -v ./...
