.PHONY: run build test clean fmt vet frontend/install frontend/dev frontend/build

# Go commands (requires frontend build)
run: frontend/build
	go run .

build: frontend/build
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
	cd frontend && npm install && cd ..

frontend/dev:
	cd frontend && npm run dev && cd ..

frontend/build:
	cd frontend && npm run build && cd ..

