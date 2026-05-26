.PHONY: build test clean install

build:
	go build -o girl ./cmd/girl/

install:
	go install ./cmd/girl/

test:
	go test ./...

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
