
export GO111MODULE=on

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

.PHONY: snapshot-release
snapshot-release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --snapshot --config deploy/.goreleaser.snapshot.yml

