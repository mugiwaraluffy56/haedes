.PHONY: build check generate lint test typecheck

build:
	pnpm build

check:
	./scripts/check-repo.sh
	pnpm lint
	pnpm test
	pnpm typecheck
	./scripts/go-test.sh
	cargo check --workspace

generate:
	./scripts/generate-contracts.sh

lint:
	pnpm lint
	./scripts/go-vet.sh
	cargo fmt --all -- --check

test:
	pnpm test
	./scripts/go-test.sh
	cargo test --workspace

typecheck:
	pnpm typecheck
