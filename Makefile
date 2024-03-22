build:
	mkdir -p bin/
	go build -o bin/server ./...

setup:
	finch compose build
	finch compose up -d
