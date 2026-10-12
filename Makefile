.PHONY: build test clean install scale scale-bench k6-corpus k6 k6-hardware

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

# React/TSX corpus built from testdata/ and examples/ fixtures.
k6-corpus:
	@bash scripts/k6-corpus.sh

# k6 against a local `girl serve` (needs k6 or Docker).
k6: build k6-corpus
	@./girl serve --root .girl-scale/k6-corpus --addr 127.0.0.1:7777 & pid=$$!; \
	trap "kill $$pid" EXIT; sleep 1; \
	if command -v k6 >/dev/null; then k6 run bench/k6/girl.js; \
	else docker run --rm --network host -v $$PWD/bench/k6:/scripts:ro grafana/k6:0.54.0 run /scripts/girl.js; fi

# k6 across simulated hardware profiles (Docker --cpus/--memory).
k6-hardware:
	@bash scripts/k6-hardware.sh

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
