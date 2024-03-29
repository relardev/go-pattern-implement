build:
	go build -o bin/go-pattern-implement

.PHONY: test
test: build
	./test/test.sh
