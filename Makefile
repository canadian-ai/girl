.PHONY: build test clean install scale scale-bench

build:
	go build -o girl ./cmd/girl/

install:
	go install ./cmd/girl/

install-full:
	@bash scripts/install.sh

test:
	go test ./...

# 10x synthetic monorepo + the full Go standard library.
scale:
	GIRL_SCALE=1 go test ./internal/commands -run TestScale -count=1 -v

# Real-world corpora (Kubernetes, VS Code, Excalidraw, ripgrep) at pinned tags.
scale-bench:
	@bash scripts/scale-bench.sh

clean:
	rm -f girl

analyze-example:
	./girl analyze examples/messy-react-form --output text

plan-example:
	./girl plan examples/messy-react-form --output markdown

pack-example:
	./girl pack examples/messy-react-form --output markdown --budget 8000

verify-example:
	./girl verify examples/messy-react-form --output text

full-example: build
	@echo "=== Analyze ==="
	./girl analyze examples/messy-react-form --output text
	@echo ""
	@echo "=== Plan ==="
	./girl plan examples/messy-react-form --output markdown
	@echo ""
	@echo "=== Verify ==="
	./girl verify examples/messy-react-form --output text
