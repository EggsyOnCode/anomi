# Order Generation Scripts

This directory contains scripts for testing the Anomi P2P network.

## generate_orders.go

A script that generates random orders and sends them to a specific node's API endpoint.

### Usage

```bash
go run generate_orders.go <node_id>
```

### Parameters

- `node_id`: The ID of the node to send orders to (e.g., "node1", "node2")

### Features

- Generates random orders with different types (MARKET, LIMIT, STOP-LIMIT)
- Supports both BUY and SELL orders
- Random quantities between 0.1 and 10.0
- Random prices based on symbol (BTC: $30k-$50k, ETH: $1k-$3k)
- Random TIF (Time In Force) for limit orders
- Sends orders continuously with 1-5 second intervals
- Automatically determines API port based on node ID

### Order Types

- **MARKET**: Orders executed immediately at market price
- **LIMIT**: Orders executed at specified price or better
- **STOP-LIMIT**: Orders with stop price and limit price

### Symbols

- BTC/USD
- ETH/USD

### Example Output

```
Starting order generation for node: node1 on port: 8081
Order sent successfully: LIMIT BUY 5.23 BTC/USD @ 42500.50
Order sent successfully: MARKET SELL 2.15 ETH/USD @ 0.00
Order sent successfully: STOP-LIMIT BUY 1.87 BTC/USD @ 38000.00
```
