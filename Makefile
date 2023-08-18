all: build run

build:
	go build .

run:
	./build/termsrs s.srs

