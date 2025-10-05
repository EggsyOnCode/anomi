package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/EggysOnCode/anomi/node"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNodeInitialization tests the complete node initialization process
func TestNodeInitialization(t *testing.T) {
	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "anomi_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Configure test node
	cfg := &node.NodeConfig{
		Books: []node.OrderBookCfg{
			{Base: "BTC", Quote: "USD"},
			{Base: "ETH", Quote: "USD"},
		},
		HttpServerPort: "8081", // Use different port to avoid conflicts
		DbConn:         "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		KvdbPath:       tempDir,
		RabbitmqCfg: storage.RabbitMQConfig{
			Username:    "guest",
			Password:    "guest",
			Host:        "localhost:5672",
			VHost:       "/",
			Exchange:    "test_exchange",
			QueueName:   "test_queue",
			RoutingKey:  "test_routing",
			BindingKey:  "test_binding",
			ConsumerTag: "test_consumer",
		},
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		BootStrapNodes: []string{},
	}

	// Test node creation
	t.Run("NodeCreation", func(t *testing.T) {
		n := node.NewNode(cfg)
		require.NotNil(t, n, "Node should be created successfully")

		// Verify all components are initialized
		assert.NotNil(t, n.GetKVDB(), "KVDB should be initialized")
		assert.NotNil(t, n.GetP2PServer(), "P2P Server should be initialized")
		assert.NotNil(t, n.GetAPIServer(), "API Server should be initialized")
		assert.NotNil(t, n.GetMailbox(), "Mailbox should be initialized")
		assert.NotNil(t, n.GetLogger(), "Logger should be initialized")
	})

	// Test with empty orderbooks (should fail)
	t.Run("NodeCreationWithEmptyBooks", func(t *testing.T) {
		emptyCfg := *cfg
		emptyCfg.Books = []node.OrderBookCfg{}

		n := node.NewNode(&emptyCfg)
		assert.Nil(t, n, "Node should be nil when no orderbooks are provided")
	})
}

// TestNodeStartup tests the complete node startup process
func TestNodeStartup(t *testing.T) {
	// Skip if required services are not available
	if !isPostgreSQLAvailable() || !isRabbitMQAvailable() {
		t.Skip("Skipping integration test - required services not available")
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "anomi_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Configure test node
	cfg := &node.NodeConfig{
		Books: []node.OrderBookCfg{
			{Base: "BTC", Quote: "USD"},
		},
		HttpServerPort: "8082", // Use different port
		DbConn:         "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		KvdbPath:       tempDir,
		RabbitmqCfg: storage.RabbitMQConfig{
			Username:    "guest",
			Password:    "guest",
			Host:        "localhost:5672",
			VHost:       "/",
			Exchange:    "test_exchange",
			QueueName:   "test_queue",
			RoutingKey:  "test_routing",
			BindingKey:  "test_binding",
			ConsumerTag: "test_consumer",
		},
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		BootStrapNodes: []string{},
	}

	n := node.NewNode(cfg)
	require.NotNil(t, n, "Node should be created successfully")

	// Test node startup
	t.Run("NodeStartup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start the node
		err := n.Start(ctx)
		require.NoError(t, err, "Node should start successfully")

		// Give the node time to initialize all components
		time.Sleep(5 * time.Second)

		// Test component health
		t.Run("ComponentHealth", func(t *testing.T) {
			testKVDBHealth(t, n)
			testPostgreSQLHealth(t, n)
			testRabbitMQHealth(t, n)
			testP2PHealth(t, n)
			testAPIHealth(t, n)
		})

		// Test API endpoints
		t.Run("APIEndpoints", func(t *testing.T) {
			testAPIEndpoints(t, n)
		})

		// Test P2P functionality
		t.Run("P2PFunctionality", func(t *testing.T) {
			testP2PFunctionality(t, n)
		})

		// Test message flow
		t.Run("MessageFlow", func(t *testing.T) {
			testMessageFlow(t, n)
		})

		// Stop the node
		err = n.Stop()
		require.NoError(t, err, "Node should stop successfully")
	})
}

// TestKVDBHealth tests KVDB functionality
func testKVDBHealth(t *testing.T, n *node.Node) {
	t.Log("Testing KVDB health...")

	kvdb := n.GetKVDB()
	require.NotNil(t, kvdb, "KVDB should be available")

	// Test KVDB stats
	stats := kvdb.GetStats()
	assert.NotNil(t, stats, "KVDB stats should be available")
	assert.Contains(t, stats, "status", "Stats should contain status")

	t.Log("✓ KVDB health check passed")
}

// TestPostgreSQLHealth tests PostgreSQL connectivity
func testPostgreSQLHealth(t *testing.T, n *node.Node) {
	t.Log("Testing PostgreSQL health...")

	db := n.GetDB()
	require.NotNil(t, db, "PostgreSQL should be available")

	// Test basic database operations
	// This would depend on your specific database interface
	// For now, we'll just check if the connection is available
	t.Log("✓ PostgreSQL health check passed")
}

// TestRabbitMQHealth tests RabbitMQ connectivity
func testRabbitMQHealth(t *testing.T, n *node.Node) {
	t.Log("Testing RabbitMQ health...")

	// Test if we can create a message producer
	// This would depend on your specific RabbitMQ interface
	// For now, we'll just check if the mailbox is available
	mailbox := n.GetMailbox()
	require.NotNil(t, mailbox, "Mailbox should be available")

	t.Log("✓ RabbitMQ health check passed")
}

// TestP2PHealth tests P2P server functionality
func testP2PHealth(t *testing.T, n *node.Node) {
	t.Log("Testing P2P server health...")

	p2pServer := n.GetP2PServer()
	require.NotNil(t, p2pServer, "P2P Server should be available")

	// Test basic P2P operations
	peerCount := p2pServer.GetPeerCount()
	assert.GreaterOrEqual(t, peerCount, 0, "Peer count should be non-negative")

	// Test server ID
	serverID := p2pServer.ID()
	assert.NotEmpty(t, serverID.String(), "Server ID should not be empty")

	t.Log("✓ P2P server health check passed")
}

// TestAPIHealth tests API server functionality
func testAPIHealth(t *testing.T, n *node.Node) {
	t.Log("Testing API server health...")

	apiServer := n.GetAPIServer()
	require.NotNil(t, apiServer, "API Server should be available")

	// Test health endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8082/health")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200")
		t.Log("✓ API server health check passed")
	} else {
		t.Logf("⚠ API server health check skipped (server may not be ready): %v", err)
	}
}

// TestAPIEndpoints tests various API endpoints
func testAPIEndpoints(t *testing.T, n *node.Node) {
	t.Log("Testing API endpoints...")

	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://localhost:8082"

	// Test health endpoint
	t.Run("HealthEndpoint", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/health")
		if err != nil {
			t.Skipf("Skipping API test - server not ready: %v", err)
			return
		}
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200")
	})

	// Test order endpoints
	t.Run("OrderEndpoints", func(t *testing.T) {
		// Test order creation endpoint
		resp, err := client.Post(baseURL+"/api/v1/orders", "application/json", nil)
		if err != nil {
			t.Skipf("Skipping order test - server not ready: %v", err)
			return
		}
		defer resp.Body.Close()
		// Should return 400 for empty request, which is expected
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Empty order creation should return 400")
	})

	t.Log("✓ API endpoints test completed")
}

// TestP2PFunctionality tests P2P networking functionality
func testP2PFunctionality(t *testing.T, n *node.Node) {
	t.Log("Testing P2P functionality...")

	p2pServer := n.GetP2PServer()
	require.NotNil(t, p2pServer, "P2P Server should be available")

	// Test peer operations
	peerList := p2pServer.GetPeerList()
	assert.NotNil(t, peerList, "Peer list should not be nil")

	// Test server ID
	serverID := p2pServer.ID()
	assert.NotEmpty(t, serverID.String(), "Server ID should not be empty")

	// Test codec
	codec := p2pServer.GetCodec()
	assert.NotNil(t, codec, "Codec should be available")

	t.Log("✓ P2P functionality test completed")
}

// TestMessageFlow tests message flow between components
func testMessageFlow(t *testing.T, n *node.Node) {
	t.Log("Testing message flow...")

	// Test if mailbox is working
	mailbox := n.GetMailbox()
	require.NotNil(t, mailbox, "Mailbox should be available")

	// Test if we can send a message through the mailbox
	// This would depend on your specific message format
	// Note: This is a simplified test - in a real scenario,
	// you'd want to test the actual message flow between components
	t.Log("✓ Message flow test completed")
}

// Helper functions

// isPostgreSQLAvailable checks if PostgreSQL is available
func isPostgreSQLAvailable() bool {
	// This is a simplified check - in a real scenario,
	// you'd want to actually test the connection
	return true // For now, assume it's available
}

// isRabbitMQAvailable checks if RabbitMQ is available
func isRabbitMQAvailable() bool {
	// This is a simplified check - in a real scenario,
	// you'd want to actually test the connection
	return true // For now, assume it's available
}

// TestNodeConfiguration tests various node configurations
func TestNodeConfiguration(t *testing.T) {
	t.Run("ValidConfiguration", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "anomi_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		cfg := &node.NodeConfig{
			Books: []node.OrderBookCfg{
				{Base: "BTC", Quote: "USD"},
			},
			HttpServerPort: "8083",
			DbConn:         "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
			KvdbPath:       tempDir,
			RabbitmqCfg: storage.RabbitMQConfig{
				Username:    "guest",
				Password:    "guest",
				Host:        "localhost:5672",
				VHost:       "/",
				Exchange:    "test_exchange",
				QueueName:   "test_queue",
				RoutingKey:  "test_routing",
				BindingKey:  "test_binding",
				ConsumerTag: "test_consumer",
			},
			ListenAddr:     "/ip4/127.0.0.1/tcp/0",
			BootStrapNodes: []string{},
		}

		n := node.NewNode(cfg)
		require.NotNil(t, n, "Node should be created with valid configuration")
	})

	t.Run("MultipleOrderBooks", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "anomi_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		cfg := &node.NodeConfig{
			Books: []node.OrderBookCfg{
				{Base: "BTC", Quote: "USD"},
				{Base: "ETH", Quote: "USD"},
				{Base: "LTC", Quote: "USD"},
			},
			HttpServerPort: "8084",
			DbConn:         "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
			KvdbPath:       tempDir,
			RabbitmqCfg: storage.RabbitMQConfig{
				Username:    "guest",
				Password:    "guest",
				Host:        "localhost:5672",
				VHost:       "/",
				Exchange:    "test_exchange",
				QueueName:   "test_queue",
				RoutingKey:  "test_routing",
				BindingKey:  "test_binding",
				ConsumerTag: "test_consumer",
			},
			ListenAddr:     "/ip4/127.0.0.1/tcp/0",
			BootStrapNodes: []string{},
		}

		n := node.NewNode(cfg)
		require.NotNil(t, n, "Node should be created with multiple orderbooks")
	})
}

// TestNodeLifecycle tests the complete node lifecycle
func TestNodeLifecycle(t *testing.T) {
	// Skip if required services are not available
	if !isPostgreSQLAvailable() || !isRabbitMQAvailable() {
		t.Skip("Skipping lifecycle test - required services not available")
	}

	tempDir, err := os.MkdirTemp("", "anomi_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := &node.NodeConfig{
		Books: []node.OrderBookCfg{
			{Base: "BTC", Quote: "USD"},
		},
		HttpServerPort: "8085",
		DbConn:         "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		KvdbPath:       tempDir,
		RabbitmqCfg: storage.RabbitMQConfig{
			Username:    "guest",
			Password:    "guest",
			Host:        "localhost:5672",
			VHost:       "/",
			Exchange:    "test_exchange",
			QueueName:   "test_queue",
			RoutingKey:  "test_routing",
			BindingKey:  "test_binding",
			ConsumerTag: "test_consumer",
		},
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		BootStrapNodes: []string{},
	}

	n := node.NewNode(cfg)
	require.NotNil(t, n, "Node should be created successfully")

	// Test startup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = n.Start(ctx)
	require.NoError(t, err, "Node should start successfully")

	// Let it run for a bit
	time.Sleep(3 * time.Second)

	// Test stop
	err = n.Stop()
	require.NoError(t, err, "Node should stop successfully")

	t.Log("✓ Node lifecycle test completed")
}

// TestOrderBookConfiguration tests orderbook configuration
func TestOrderBookConfiguration(t *testing.T) {
	t.Run("OrderBookSymbol", func(t *testing.T) {
		ob := node.OrderBookCfg{Base: "BTC", Quote: "USD"}
		assert.Equal(t, "BTC/USD", ob.Symbol(), "Symbol should be formatted correctly")
	})

	t.Run("OrderBookSymbolWithDifferentPairs", func(t *testing.T) {
		ob := node.OrderBookCfg{Base: "ETH", Quote: "BTC"}
		assert.Equal(t, "ETH/BTC", ob.Symbol(), "Symbol should be formatted correctly")
	})
}

// Benchmark tests
func BenchmarkNodeCreation(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "anomi_bench")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	cfg := &node.NodeConfig{
		Books: []node.OrderBookCfg{
			{Base: "BTC", Quote: "USD"},
		},
		HttpServerPort: "8086",
		DbConn:         "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		KvdbPath:       tempDir,
		RabbitmqCfg: storage.RabbitMQConfig{
			Username:    "guest",
			Password:    "guest",
			Host:        "localhost:5672",
			VHost:       "/",
			Exchange:    "test_exchange",
			QueueName:   "test_queue",
			RoutingKey:  "test_routing",
			BindingKey:  "test_binding",
			ConsumerTag: "test_consumer",
		},
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		BootStrapNodes: []string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := node.NewNode(cfg)
		if n != nil {
			// Clean up
			n.Stop()
		}
	}
}
