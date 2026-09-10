.PHONY: dev dev-client dev-server build build-client build-server clean start

# Default target
all: build

# Run both client and server in development mode simultaneously
dev:
	@make -j2 dev-client dev-server

dev-client:
	cd client && npm run dev

dev-server:
	cd server && go run cmd/retriva/main.go

# Build the application for production
build: build-client build-server

build-client:
	cd client && npm run build

build-server:
	mkdir -p bin
	cd server && go build -o ../bin/retriva cmd/retriva/main.go

# Start the production application (must be built first)
start:
	./bin/retriva

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf client/dist/
