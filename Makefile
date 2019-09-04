
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

.PHONY: test
test:
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: unfork
unfork: fmt vet
	go build -o bin/unfork github.com/replicatedhq/unfork/cmd/unfork

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
