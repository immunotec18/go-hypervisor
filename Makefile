BIN ?= hv
ENTITLEMENTS ?= hypervisor.entitlements
CODESIGN_ID ?= -

default: dev

.PHONY: bump
bump:
	@echo "🚀 Bumping Version"
	git tag $(shell svu patch)
	git push --tags

.PHONY: build
build:
	@echo "🚀 Building Version $(shell svu current)"
	cd cmd/hv && go build -o ../../$(BIN) .

.PHONY: sign
sign: build
	@echo "🔐 Codesigning $(BIN) with $(ENTITLEMENTS)"
	codesign --sign $(CODESIGN_ID) --force --entitlements=$(ENTITLEMENTS) $(BIN)

.PHONY: verify
verify: $(BIN)
	@echo "🔍 Verifying entitlements for $(BIN)"
	codesign -dv --entitlements - $(BIN) 2>&1 | sed -n '1,120p'

.PHONY: dev
dev: build sign verify
	@echo "✅ Dev build ready: $(BIN)"

.PHONY: test
test:
	@echo "🧪 Running integration tests"
	@hack/make/test_integration

.PHONY: clean
clean:
	@echo "🧹 Cleaning up"
	rm -f $(BIN)

.PHONY: release
release:
	@echo "🚀 Releasing Version $(shell svu current)"
	goreleaser build --id default --clean --snapshot --single-target --output dist/hv

.PHONY: snapshot
snapshot:
	@echo "🚀 Snapshot Version $(shell svu current)"
	goreleaser --clean --timeout 60m --snapshot

.PHONY: vhs
vhs: release
	@echo "📼 VHS Recording"
	@echo "Please ensure you have the 'vhs' command installed."
	vhs < vhs.tape	
