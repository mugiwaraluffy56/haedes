.PHONY: build check format format-check generate generate-write lint test typecheck

build:
	pnpm build

check:
	./scripts/check-repo.sh
	./scripts/generate-contracts.sh --check
	pnpm lint
	pnpm test
	pnpm typecheck
	./scripts/go-test.sh
	cargo check --workspace

format:
	cargo fmt --all

format-check:
	cargo fmt --all -- --check

generate:
	./scripts/generate-contracts.sh --check

generate-write:
	./scripts/generate-contracts.sh --write

lint:
	pnpm lint
	./scripts/go-vet.sh
	$(MAKE) format-check

test:
	pnpm test
	./scripts/go-test.sh
	cargo test --workspace

typecheck:
	pnpm typecheck
