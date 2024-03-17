build:
	go build -o bin/go-component-generator

.PHONY: test
test: build
	./test/test.sh
