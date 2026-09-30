.PHONY: build test demo

build:
	go build -trimpath -o bin/instagram-scraper ./cmd/instagram-scraper

test:
	python3 -m unittest discover -s tests -v
	go test -race ./...
	go vet ./...

demo: build
	./bin/instagram-scraper --demo
