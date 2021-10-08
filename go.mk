GO         ?= go
LINTER     ?= golangci-lint
GO_TESTSUM ?= gotestsum
GIT_DIRTY  := $(shell git diff --quiet || echo '-dirty')
VERSION	   := $(shell [ -z $(git tag --points-at HEAD) ] || echo $(git tag --points-at HEAD))
COMMIT     := $(shell git rev-parse --short HEAD)$(GIT_DIRTY)
LDFLAGS    += -ldflags '-extldflags "-static" -s -w -X=main.GitTag=$(VERSION) -X=main.GitCommit=$(COMMIT)' # -s -w reduces binary size by removing some debug information
BUILDFLAGS += -installsuffix cgo --tags release

BUILD_PATH ?= $(shell pwd)
CMD = $(BUILD_PATH)/o-neko-url-trigger
CMD_SRC = cmd/o-neko-url-trigger/*.go

all: build lint

.PHONY: build test lint clean build

clean:
	rm -f $(CMD)

run:
	$(GO) run $(LDFLAGS) $(CMD_SRC) $(ARGS)

test:
	$(GO) test -v ./pkg/**/* -coverprofile cover.out

test-ci:
	$(GO_TESTSUM) --format testname --junitfile test_results.xml -- -v ./pkg/**/* -coverprofile cover.out

lint:
	$(GO) mod verify
	$(LINTER) run -v --no-config --deadline=5m

prepare:
	$(GO) mod download

build:
	$(GO) build -o $(CMD) -a $(BUILDFLAGS) $(LDFLAGS) $(CMD_SRC)
	upx $(CMD) # reduce binary size

build-for-docker:
	CGO_ENABLED=0 GOOS=linux $(GO) build -o $(CMD) -a $(BUILDFLAGS) $(LDFLAGS) $(CMD_SRC)
	upx $(CMD) # reduce binary size
