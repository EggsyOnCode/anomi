build:
	go build -o ./bin/anomi

run: build
	./bin/anomi

test: 
	go test -v ./... -cover

# Run the P2P test with two nodes and order generation
test-p2p:
	go run main.go

# Generate orders for a specific node
generate-orders:
	@if [ -z "$(NODE_ID)" ]; then \
		echo "Usage: make generate-orders NODE_ID=node1"; \
		exit 1; \
	fi
	go run scripts/generate_orders.go $(NODE_ID)
