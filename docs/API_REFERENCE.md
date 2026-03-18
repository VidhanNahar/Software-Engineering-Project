# Stock Market Backend - Quick API Reference

## Base URL
```
http://localhost:8080/api
```

## Authentication
Add header to protected endpoints:
```
Authorization: Bearer {access_token}
```

## Common Endpoints

### Register
```
POST /auth/register
{
  "name": "John",
  "email_id": "john@example.com",
  "password": "pass123",
  "date_of_birth": "1990-05-15"
}
```

### Login
```
POST /auth/login
{
  "email_id": "john@example.com",
  "password": "pass123"
}
```

### Get All Stocks
```
GET /stocks
```

### Buy Stocks
```
POST /transactions/buy
Authorization: Bearer {token}
{
  "stock_id": "uuid",
  "quantity": 10,
  "price": 150.50
}
```

### Sell Stocks
```
POST /transactions/sell
Authorization: Bearer {token}
{
  "stock_id": "uuid",
  "quantity": 5,
  "price": 160.00
}
```

### Get Portfolio
```
GET /portfolio
Authorization: Bearer {token}
```

### Get Wallet
```
GET /wallet
Authorization: Bearer {token}
```

### Get Transaction History
```
GET /transactions/history?limit=50&offset=0
Authorization: Bearer {token}
```

### Add to Watchlist
```
POST /watchlist
Authorization: Bearer {token}
{
  "stock_id": "uuid"
}
```

### Get Watchlist
```
GET /watchlist
Authorization: Bearer {token}
```

## Status Codes
- 200: OK
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 404: Not Found
- 500: Internal Error

## Error Examples
- "insufficient balance: required 280,075.00, available 98,200.00"
- "insufficient stock holdings: required 5, available 0"
- "invalid credentials"

See BACKEND_GUIDE.md for complete documentation.
