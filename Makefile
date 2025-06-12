.PHONY: run build test clean fmt vet frontend/install frontend/dev frontend/build

# Go commands
run:
	go run .

build:
	go build -o s3c .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f s3c
	rm -rf frontend/dist

# Frontend commands
frontend/install:
	cd frontend && npm install

frontend/dev:
	cd frontend && npm run dev

frontend/build:
	cd frontend && npm run build

# Combined build (frontend + backend)
build-all: frontend/build build