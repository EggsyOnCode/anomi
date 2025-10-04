# Order API Examples

This document provides examples of how to use the Order Creation and Update API endpoints.

## Order Creation

### Market Order (Base Quantity)
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "orderType": "MARKET",
    "userID": "user123",
    "side": 0,
    "quantity": "100.50",
    "isQuote": false
  }'
```

### Market Order (Quote Quantity)
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "orderType": "MARKET",
    "userID": "user123",
    "side": 1,
    "quantity": "1000.00",
    "isQuote": true
  }'
```

### Limit Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "orderType": "LIMIT",
    "userID": "user123",
    "side": 0,
    "quantity": "50.25",
    "price": "1000.00",
    "tif": "GTC",
    "isQuote": false
  }'
```

### Stop-Limit Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "orderType": "STOP-LIMIT",
    "userID": "user123",
    "side": 1,
    "quantity": "25.75",
    "price": "950.00",
    "stop": "1000.00",
    "isQuote": false
  }'
```

## Order Update

### Update Quantity Only
```bash
curl -X PUT http://localhost:8080/api/v1/orders/{orderID} \
  -H "Content-Type: application/json" \
  -d '{
    "quantity": "200.00"
  }'
```

### Update Price Only
```bash
curl -X PUT http://localhost:8080/api/v1/orders/{orderID} \
  -H "Content-Type: application/json" \
  -d '{
    "price": "1050.00"
  }'
```

### Update Multiple Fields
```bash
curl -X PUT http://localhost:8080/api/v1/orders/{orderID} \
  -H "Content-Type: application/json" \
  -d '{
    "quantity": "150.00",
    "price": "1100.00",
    "stop": "1050.00"
  }'
```

## Field Descriptions

### Order Types
- `MARKET`: Execute immediately at current market price
- `LIMIT`: Execute only at specified price or better
- `STOP-LIMIT`: Stop order that becomes a limit order when triggered

### Sides
- `0`: BUY order
- `1`: SELL order

### TIF (Time in Force)
- `GTC`: Good Till Canceled
- `IOC`: Immediate or Cancel
- `FOK`: Fill or Kill

### Required Fields by Order Type

#### MARKET Orders
- `orderType`: "MARKET"
- `userID`: User identifier
- `side`: 0 (BUY) or 1 (SELL)
- `quantity`: Order quantity

#### LIMIT Orders
- `orderType`: "LIMIT"
- `userID`: User identifier
- `side`: 0 (BUY) or 1 (SELL)
- `quantity`: Order quantity
- `price`: Limit price
- `tif`: Time in force

#### STOP-LIMIT Orders
- `orderType`: "STOP-LIMIT"
- `userID`: User identifier
- `side`: 0 (BUY) or 1 (SELL)
- `quantity`: Order quantity
- `price`: Limit price
- `stop`: Stop price

### Optional Fields
- `isQuote`: Whether quantity is in quote currency (default: false)
- `oco`: One-Cancels-Other reference ID

## Response Format

All responses follow this format:

```json
{
  "success": true,
  "message": "Order created successfully",
  "data": {
    "id": "order-uuid",
    "orderType": "LIMIT",
    "userID": "user123",
    "side": "BUY",
    "isQuote": false,
    "quantity": "100.50",
    "originalQty": "100.50",
    "price": "1000.00",
    "stop": "0",
    "canceled": false,
    "role": "TAKER",
    "tif": "GTC",
    "oco": "",
    "createdAt": "2024-01-01T12:00:00Z",
    "updatedAt": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Error Responses

Error responses follow this format:

```json
{
  "success": false,
  "error": "Validation failed",
  "message": "Order ID is required",
  "timestamp": "2024-01-01T12:00:00Z"
}
```
