all: build run

build:
	go build .

run:
	./termsrs s.srs

