package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/EggysOnCode/anomi/node"
	"github.com/EggysOnCode/anomi/storage"
)

// CreateNode creates a new node with the specified configuration
func CreateNode(nodeID, httpPort, listenAddr, kvdbPath, dbConn string, books []node.OrderBookCfg, bootstrapNodes []string, rabbitmqPort string) *node.Node {
	cfg := &node.NodeConfig{
		Books:          books,
		HttpServerPort: httpPort,
		DbConn:         dbConn,
		KvdbPath:       kvdbPath,
		RabbitmqCfg: storage.RabbitMQConfig{
			Username:    "guest",
			Password:    "guest",
			Host:        rabbitmqPort, // "localhost:5672"
			VHost:       "/",
			Exchange:    "test_exchange",
			QueueName:   "test_queue",
			RoutingKey:  "",
			BindingKey:  "", // Make binding key same as routing key
			ConsumerTag: "test_consumer",
		},
		ListenAddr:     listenAddr,
		BootStrapNodes: bootstrapNodes,
	}

	return node.NewNode(cfg)
}

// runOrderScript runs the order generation script for a specific node
func runOrderScript(nodeID string) error {
	scriptPath := "./scripts/generate_orders.go"

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("order generation script not found at %s", scriptPath)
	}

	cmd := exec.Command("go", "run", scriptPath, nodeID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {
	// Create temporary directories for test data
	tempDir1, err := os.MkdirTemp("", "anomi_test_node1")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "anomi_test_node2")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir2)

	// Create scripts directory if it doesn't exist
	scriptsDir := "./scripts"
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		panic(err)
	}

	// Define order books
	books := []node.OrderBookCfg{
		{Base: "BTC", Quote: "USD"},
		{Base: "ETH", Quote: "USD"},
	}

	// Create first node
	fmt.Println("Creating Node 1...")
	node1 := CreateNode(
		"node1",
		"8081",
		"localhost:9001",
		tempDir1,
		"postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		books,
		[]string{"localhost:9000"},
		"localhost:5672",
	)
	if node1 == nil {
		fmt.Println("Failed to create Node 1. Continuing without Node 1.")
	}

	// Create second node
	fmt.Println("Creating Node 2...")
	node2 := CreateNode(
		"node2",
		"8082",
		"localhost:9000",
		tempDir2,
		"postgres://testuser:testpass@localhost:5433/testdb1?sslmode=disable",
		books,
		[]string{"localhost:9001"},
		"localhost:5673",
	)
	if node2 == nil {
		fmt.Println("Failed to create Node 2. Continuing without Node 2.")
	}

	// Start both nodes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start node 1
	if node1 != nil {
		go func() {
			if err := node1.Start(ctx); err != nil {
				fmt.Printf("Node 1 error: %v\n", err)
			}
		}()
	}

	// Wait a bit for node 1 to start its discovery service
	time.Sleep(2 * time.Second)

	// Start node 2
	if node2 != nil {
		go func() {
			if err := node2.Start(ctx); err != nil {
				fmt.Printf("Node 2 error: %v\n", err)
			}
		}()
	}

	// Wait a bit for nodes to start
	time.Sleep(3 * time.Second)

	// Run order generation script for node 1
	if node1 != nil {
		fmt.Println("Starting order generation for Node 1...")
		go func() {
			if err := runOrderScript("node1"); err != nil {
				fmt.Printf("Order script error: %v\n", err)
			}
		}()
	}

	// Run order generation script for node 2
	if node2 != nil {
		fmt.Println("Starting order generation for Node 2...")
		go func() {
			if err := runOrderScript("node2"); err != nil {
				fmt.Printf("Order script error: %v\n", err)
			}
		}()
	}

	// Keep running until interrupted
	fmt.Println("Nodes started. Press Ctrl+C to stop...")
	select {}
}
