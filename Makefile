.PHONY: build check check-core check-contracts check-containers check-terraform check-security \
	check-dependencies check-secrets format format-check generate generate-write lint test typecheck

build:
	pnpm build

check:
	$(MAKE) check-core
	$(MAKE) check-contracts
	$(MAKE) check-containers
	$(MAKE) check-terraform
	$(MAKE) check-security

check-core:
	./scripts/check-core.sh

check-contracts:
	./scripts/check-contracts.sh

check-containers:
	./scripts/check-containers.sh

check-terraform:
	./scripts/check-terraform.sh

check-security:
	./scripts/check-security.sh

check-dependencies:
	./scripts/check-dependencies.sh

check-secrets:
	./scripts/check-secrets.sh

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
