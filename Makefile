
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

SHELL := /bin/bash -o pipefail
SRC = $(shell find pkg cmd -name "*.go")

VERSION_PACKAGE = github.com/replicatedhq/unfork/pkg/version
VERSION ?=`git describe --tags`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

GIT_TREE = $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifneq "$(GIT_TREE)" ""
define GIT_UPDATE_INDEX_CMD
git update-index --assume-unchanged
endef
define GIT_SHA
`git rev-parse HEAD`
endef
else
define GIT_UPDATE_INDEX_CMD
echo "Not a git repo, skipping git update-index"
endef
define GIT_SHA
""
endef
endif

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

.PHONY: test
test:
	go test -cover ./pkg/... ./cmd/...

cover.out: $(SRC)
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: unfork
unfork: fmt vet
	go build ${LDFLAGS} -o bin/unfork github.com/replicatedhq/unfork/cmd/unfork

.PHONY: index
index: unfork
	./bin/unfork index

.PHONY: ensureindex
ensureindex: unfork
ifeq (,$(wildcard ./bin/charts.json))
	./bin/unfork index
endif

.PHONY: run
run: unfork ensureindex
	./bin/unfork

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: lint
lint:
	golangci-lint run pkg/... cmd/...

.PHONY: release
release: export GITHUB_TOKEN = $(shell echo ${GITHUB_TOKEN_REPLICATEDBOT})
release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --config deploy/.goreleaser.yml
