# Technical User Flow Documentation

## System Overview

This is a **Stock Trading Platform** that enables users to register, authenticate, manage portfolios, execute buy/sell transactions, track stocks, and manage their trading wallet. The application follows a modern full-stack architecture with a React frontend and Go backend, using TimescaleDB (PostgreSQL extension) for time-series data storage and Redis for token management.

---

## Technology Stack

### Frontend
- **Framework**: React 18.3.1 with TypeScript
- **Build Tool**: Vite 7.3.1
- **Routing**: React Router 7.13.1
- **UI Libraries**:
  - Material-UI (MUI) 7.3.5
  - Radix UI primitives
  - Tailwind CSS 4.2.1
- **State Management**: React Context API (ThemeContext)
- **Charts**: Chart.js 4.5.1, Recharts 2.15.2
- **HTTP Client**: Fetch API (browser native)
- **Notifications**: Sonner 2.0.3

### Backend
- **Language**: Go (Golang)
- **Web Framework**: Gorilla Mux (routing)
- **Database**: TimescaleDB (PostgreSQL 15 with time-series extensions)
- **Caching/Session**: Redis (Alpine)
- **Authentication**: JWT (golang-jwt/jwt v5)
- **Password Hashing**: bcrypt (golang.org/x/crypto/bcrypt)

### Infrastructure
- **Containerization**: Docker Compose
- **Services**:
  - TimescaleDB on port 5432
  - Redis on port 6379
  - Go backend on port 8080 (default)
  - React frontend on port 5173 (Vite dev server)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                          │
│  (React SPA - Browser)                                       │
│  - Login/Dashboard/Portfolio/Trade/Market/StockDetails       │
└──────────────────┬──────────────────────────────────────────┘
                   │ HTTP/HTTPS (REST API)
                   │ Authorization: Bearer <JWT>
┌──────────────────▼──────────────────────────────────────────┐
│                     API GATEWAY LAYER                        │
│  Go HTTP Server (Gorilla Mux Router)                         │
│  - Public Routes (no auth)                                   │
│  - Protected Routes (JWT middleware)                         │
└──────────────────┬──────────────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
┌───────▼────────┐   ┌────────▼─────────┐
│ HANDLER LAYER  │   │  AUTH MIDDLEWARE │
│ - auth.go      │   │  - JWT validate  │
│ - stock.go     │   │  - UserID inject │
│ - transaction  │   └──────────────────┘
└───────┬────────┘
        │
┌───────▼────────┐
│ SERVICE LAYER  │
│ - transaction  │
│   service      │
│ - business     │
│   logic        │
└───────┬────────┘
        │
        ├──────────┬─────────────┐
        │          │             │
┌───────▼─────┐ ┌──▼─────┐ ┌────▼────┐
│ TimescaleDB │ │ Redis  │ │  Model  │
│ (Postgres)  │ │ Cache  │ │  Layer  │
│ - users     │ │ - JWT  │ │ - User  │
│ - stock     │ │   BL   │ │ - Stock │
│ - orders    │ └────────┘ │ - Trans │
│ - portfolio │            └─────────┘
│ - wallet    │
│ - watchlist │
└─────────────┘
```

---

## Complete User Journey

### 1. User Registration & Onboarding

#### Frontend Flow
1. User navigates to `/login` (Login.tsx)
2. Clicks on "Register" or "Sign Up"
3. Fills registration form with:
   - Name
   - Email
   - Password
   - Aadhar ID (optional)
   - PAN ID (optional)
   - Phone Number (optional)
   - Date of Birth
   - Email verification status

#### Backend Processing
**Endpoint**: `POST /api/auth/register`

**Request Body**:
```json
{
  "name": "John Doe",
  "email_id": "john@example.com",
  "password": "SecurePass123",
  "aadhar_id": "123456789012",
  "pan_id": "ABCDE1234F",
  "phone_number": "9876543210",
  "date_of_birth": "1990-01-01",
  "is_verified_email": false
}
```

**Processing Steps** (handler/auth.go:45-100):
1. Validate required fields (name, email, password, DOB)
2. Parse date_of_birth to `time.Time`
3. Hash password using bcrypt with default cost (10)
4. Insert user into `users` table with UUID generation
5. Generate access token (15-min expiry) and refresh token (7-day expiry)
6. Return user_id and both tokens

**Database Transaction**:
```sql
INSERT INTO users (name, email_id, password, aadhar_id, pan_id, phone_number, date_of_birth, is_verified_email)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING user_id
```

**Response**:
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc..."
}
```

**Wallet Initialization** (main.go:99-134):
- System checks for users without wallets on startup
- Creates wallet with initial balance of ₹100,000 for trading
- Uses `service.InitializeWallet(ctx, userID, 100000.0)`

---

### 2. User Authentication

#### Frontend Flow
1. User enters email and password on Login page
2. Form submits to backend
3. On success, stores tokens in localStorage
4. Sets `isLoggedIn` flag to "true"
5. Redirects to Dashboard (`/`)

#### Backend Processing
**Endpoint**: `POST /api/auth/login`

**Request**:
```json
{
  "email_id": "john@example.com",
  "password": "SecurePass123"
}
```

**Processing Steps** (handler/auth.go:109-153):
1. Query user by email
2. Retrieve user_id and hashed password
3. Compare password hash using bcrypt
4. On success, generate new access and refresh tokens
5. Return tokens

**Token Structure** (auth/jwt.go):
- **Access Token**: HS256 signed, 15-minute expiry
- **Refresh Token**: HS256 signed, 7-day expiry
- **Payload**: `{"user_id": "uuid", "exp": timestamp, "iat": timestamp}`

**Token Storage**:
- Frontend: localStorage
- Backend: Refresh tokens can be blacklisted in Redis

---

### 3. Session Management

#### Token Refresh Flow
**Endpoint**: `POST /api/auth/refresh`

**Request**:
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Processing** (handler/auth.go:161-191):
1. Check Redis blacklist (`bl:` prefix) for revoked tokens
2. Validate refresh token signature and expiry
3. Extract user_id from claims
4. Generate new access token (15-min)
5. Return new access token

#### Logout Flow
**Endpoint**: `POST /api/auth/logout`

**Processing** (handler/auth.go:199-219):
1. Validate refresh token
2. Calculate TTL until natural expiry
3. Add token to Redis blacklist with key `bl:{refresh_token}`
4. Set Redis expiry to match token expiry
5. Frontend clears localStorage and redirects to `/login`

---

### 4. Dashboard & Market Overview

#### Frontend Components
- **Dashboard.tsx**: Main landing page after login
- **MarketOverview.tsx**: Stock market statistics
- **Navigation**: Layout.tsx with sidebar

#### Backend Data Sources
**Endpoint**: `GET /api/stocks`

**Processing** (handler/stock.go:54-90):
1. Query all stocks from database
2. Order by name ascending
3. Return stocks with metadata

**Response**:
```json
{
  "stocks": [
    {
      "stock_id": "uuid",
      "symbol": "AAPL",
      "name": "Apple Inc.",
      "price": 175.50,
      "quantity": 1000,
      "timestamp": "2026-03-18T10:30:00Z"
    }
  ],
  "count": 150
}
```

**Stock Search** (`GET /api/stocks/search?q=apple`):
- Case-insensitive ILIKE search on name and symbol
- Limited to 20 results
- Returns matching stocks

---

### 5. Stock Details & Analysis

#### Frontend Flow
1. User clicks on stock from dashboard
2. Navigate to `/stock/:symbol` (StockDetails.tsx)
3. Display detailed stock information

#### Backend Endpoints

**Get Stock by ID**: `GET /api/stocks/{stock_id}`
- Returns single stock details
- Includes price, quantity, timestamp

**Get Stock Statistics**: `GET /api/stocks/{stock_id}/stats`

**Processing** (handler/stock.go:472-524):
1. Fetch stock information
2. Calculate trading volumes:
   ```sql
   SELECT
     COALESCE(SUM(CASE WHEN o.quantity > 0 THEN o.quantity ELSE 0 END), 0) as buy_volume,
     COALESCE(SUM(CASE WHEN o.quantity < 0 THEN o.quantity ELSE 0 END), 0) as sell_volume
   FROM orders o
   WHERE o.stock_id = $1 AND o.status = 'Filed'
   ```
3. Return stock + volume metrics

---

### 6. Buying Stocks

#### Frontend Flow
1. User navigates to Trade page (Trade.tsx)
2. Selects stock, enters quantity and price
3. Submits buy order

#### Backend Processing
**Endpoint**: `POST /api/transactions/buy` (Protected)

**Request**:
```json
{
  "stock_id": "uuid",
  "quantity": 10,
  "price": 175.50
}
```

**Service Layer Transaction** (service/transaction.go:13-107):

1. **Calculate Total Cost**:
   ```go
   totalCost = quantity × currentPrice
   ```

2. **Start Database Transaction** (ACID compliance):
   ```go
   tx, _ := db.Pool.Begin(ctx)
   defer tx.Rollback(ctx)
   ```

3. **Lock User Wallet** (pessimistic locking):
   ```sql
   SELECT balance, locked_balance
   FROM wallet
   WHERE user_id = $1
   FOR UPDATE
   ```

4. **Validate Sufficient Balance**:
   ```go
   availableBalance := balance - lockedBalance
   if availableBalance < totalCost {
     return error
   }
   ```

5. **Create Order Record**:
   ```sql
   INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
   VALUES ($1, $2, $3, 'Filed', $4, $5)
   RETURNING order_id
   ```

6. **Lock Fund in Wallet**:
   ```sql
   UPDATE wallet
   SET locked_balance = locked_balance + $totalCost
   WHERE user_id = $userID
   ```

7. **Update Portfolio**:
   - Check if user already holds stock
   - If yes: Insert new portfolio entry with same portfolio_id
   - If no: Create new portfolio entry
   ```sql
   INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
   VALUES ($1, $2, $3, $4, $5)
   ```

8. **Commit Transaction**:
   ```go
   tx.Commit(ctx)
   ```

**Response**:
```json
{
  "order_id": "uuid",
  "status": "Filed",
  "message": "Stock purchase successful",
  "remaining_balance": 95000.00,
  "total_cost": 1755.00,
  "quantity": 10,
  "price_per_stock": 175.50
}
```

#### Error Scenarios
- Insufficient balance → 400 Bad Request
- Invalid stock_id → 400 Bad Request
- Database failure → Transaction rollback + 500 Error

---

### 7. Selling Stocks

#### Frontend Flow
1. User views Portfolio (Portfolio.tsx)
2. Selects stock to sell
3. Enters quantity and price
4. Submits sell order

#### Backend Processing
**Endpoint**: `POST /api/transactions/sell` (Protected)

**Request**:
```json
{
  "stock_id": "uuid",
  "quantity": 5,
  "price": 180.00
}
```

**Service Layer Transaction** (service/transaction.go:110-184):

1. **Calculate Proceeds**:
   ```go
   totalProceeds = quantity × currentPrice
   ```

2. **Start Database Transaction**

3. **Validate Holdings**:
   ```sql
   SELECT COALESCE(SUM(quantity), 0)
   FROM portfolio
   WHERE user_id = $1 AND stock_id = $2
   ```
   - Must have sufficient quantity to sell

4. **Create Sell Order**:
   ```sql
   INSERT INTO orders (stock_id, user_id, timestamp, status, quantity, price_per_stock)
   VALUES ($1, $2, $3, 'Filed', $4, $5)
   ```

5. **Credit Wallet**:
   ```sql
   UPDATE wallet
   SET balance = balance + $totalProceeds
   WHERE user_id = $userID
   ```

6. **Record Sell in Portfolio** (negative quantity):
   ```sql
   INSERT INTO portfolio (user_id, stock_id, transaction_time, price, quantity)
   VALUES ($1, $2, $3, $4, -$5)
   ```

7. **Commit Transaction**

**Response**:
```json
{
  "order_id": "uuid",
  "status": "Filed",
  "message": "Stock sale successful",
  "remaining_balance": 95900.00,
  "total_proceeds": 900.00,
  "quantity": 5,
  "price_per_stock": 180.00
}
```

---

### 8. Portfolio Management

#### Frontend Flow
1. User navigates to Portfolio page (Portfolio.tsx)
2. View all current holdings
3. See profit/loss calculations

#### Backend Processing
**Endpoint**: `GET /api/portfolio` (Protected)

**Database Query** (service/transaction.go:187-233):
```sql
SELECT DISTINCT
  p.portfolio_id,
  p.user_id,
  p.stock_id,
  s.symbol,
  s.name,
  COALESCE(SUM(p.quantity), 0),
  AVG(p.price),
  s.price,
  p.transaction_time
FROM portfolio p
JOIN stock s ON p.stock_id = s.stock_id
WHERE p.user_id = $1
GROUP BY p.stock_id, p.portfolio_id, p.user_id, s.symbol, s.name, s.price, p.transaction_time
HAVING COALESCE(SUM(p.quantity), 0) > 0
```

**Processing**:
- Aggregates all buy/sell transactions per stock
- Calculates average buy price
- Fetches current stock price
- Computes total value: `quantity × currentPrice`

**Response**:
```json
{
  "holdings": [
    {
      "portfolio_id": "uuid",
      "user_id": "uuid",
      "stock_id": "uuid",
      "stock_symbol": "AAPL",
      "stock_name": "Apple Inc.",
      "quantity": 5,
      "average_buy_price": 175.50,
      "current_price": 180.00,
      "total_value": 900.00,
      "transaction_time": "2026-03-18T..."
    }
  ],
  "count": 3
}
```

---

### 9. Wallet Management

#### Frontend Flow
1. Dashboard displays wallet balance
2. User can view detailed wallet info

#### Backend Processing
**Endpoint**: `GET /api/wallet` (Protected)

**Database Query** (service/transaction.go:236-249):
```sql
SELECT wallet_id, user_id, balance, locked_balance
FROM wallet
WHERE user_id = $1
```

**Calculation**:
```go
availableBalance = balance - lockedBalance
```

**Response**:
```json
{
  "wallet_id": "uuid",
  "user_id": "uuid",
  "balance": 100000.00,
  "locked_balance": 1755.00,
  "available_balance": 98245.00
}
```

**Balance Types**:
- **balance**: Total funds (includes locked)
- **locked_balance**: Funds allocated to pending orders
- **available_balance**: Funds available for trading

---

### 10. Transaction History

#### Frontend Flow
1. User views transaction history
2. Paginated list of all orders

#### Backend Processing
**Endpoint**: `GET /api/transactions/history?limit=50&offset=0` (Protected)

**Query Parameters**:
- `limit`: Max records (default: 50, max: 500)
- `offset`: Pagination offset (default: 0)

**Database Query** (service/transaction.go:252-286):
```sql
SELECT order_id, user_id, stock_id, timestamp, status, quantity, price_per_stock
FROM orders
WHERE user_id = $1
ORDER BY timestamp DESC
LIMIT $2 OFFSET $3
```

**Response**:
```json
{
  "transactions": [
    {
      "order_id": "uuid",
      "user_id": "uuid",
      "stock_id": "uuid",
      "type": "buy",
      "quantity": 10,
      "price_per_stock": 175.50,
      "total_amount": 1755.00,
      "status": "Filed",
      "timestamp": "2026-03-18T10:30:00Z"
    }
  ],
  "count": 15,
  "limit": 50,
  "offset": 0
}
```

---

### 11. Watchlist Management

#### Adding to Watchlist
**Endpoint**: `POST /api/watchlist` (Protected)

**Request**:
```json
{
  "stock_id": "uuid"
}
```

**Processing** (handler/stock.go:308-374):
1. Validate user authentication
2. Fetch stock details
3. Insert into watchlist table
4. Return watchlist item

#### Viewing Watchlist
**Endpoint**: `GET /api/watchlist` (Protected)

**Query**:
```sql
SELECT w.watchlist_id, w.user_id, w.stock_id, s.symbol, s.name, s.price, w.timestamp
FROM watchlist w
JOIN stock s ON w.stock_id = s.stock_id
WHERE w.user_id = $1
ORDER BY w.timestamp DESC
```

#### Removing from Watchlist
**Endpoint**: `DELETE /api/watchlist/{watchlist_id}` (Protected)

**Processing**:
1. Verify ownership (user_id match)
2. Delete record
3. Return success message

---

### 12. Admin Operations (Stock Management)

#### Create Stock
**Endpoint**: `POST /api/admin/stocks` (Protected)

**Request**:
```json
{
  "symbol": "GOOGL",
  "name": "Alphabet Inc.",
  "price": 140.50,
  "quantity": 5000
}
```

**Processing** (handler/stock.go:179-222):
1. Verify authentication (future: admin role check)
2. Validate inputs
3. Insert stock with UUID generation
4. Return created stock

#### Update Stock
**Endpoint**: `PUT /api/admin/stocks/{stock_id}` (Protected)

**Request**:
```json
{
  "price": 142.00,
  "quantity": 4800
}
```

**Processing**:
1. Update stock price and quantity
2. Update timestamp to current time

#### Delete Stock
**Endpoint**: `DELETE /api/admin/stocks/{stock_id}` (Protected)

**Processing**:
1. Verify stock exists
2. Delete stock record
3. Note: Foreign key constraints may prevent deletion if referenced

---

## Database Schema Details

### Users Table
```sql
CREATE TABLE users (
    user_id           UUID PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    email_id          VARCHAR(255) UNIQUE,
    password          VARCHAR(255) NOT NULL,
    aadhar_id         NUMERIC(12,0) UNIQUE,
    pan_id            VARCHAR(10) UNIQUE,
    phone_number      NUMERIC(10,0) UNIQUE,
    date_of_birth     DATE NOT NULL,
    is_verified_email BOOLEAN DEFAULT FALSE
)
```

### Stock Table
```sql
CREATE TABLE stock (
    stock_id  UUID PRIMARY KEY,
    name      VARCHAR(64) NOT NULL,
    symbol    VARCHAR(10) NOT NULL,  -- Added in code
    price     NUMERIC(15,2) CHECK (price > 0),
    timestamp TIMESTAMPTZ NOT NULL,
    quantity  INTEGER CHECK (quantity >= 0)
)
```

### Portfolio Table (Time-Series Design)
```sql
CREATE TABLE portfolio (
    portfolio_id     UUID NOT NULL,
    user_id          UUID REFERENCES users(user_id),
    stock_id         UUID REFERENCES stock(stock_id),
    transaction_time TIMESTAMPTZ NOT NULL,
    price            NUMERIC(15,2) CHECK (price > 0),
    quantity         INTEGER,  -- Can be negative for sells
    PRIMARY KEY (portfolio_id, stock_id, transaction_time)
)
```

**Design Pattern**: Append-only time-series
- Each buy/sell creates new row
- Portfolio balance: `SUM(quantity)` per stock
- Average cost: `AVG(price)` where quantity > 0

### Orders Table
```sql
CREATE TABLE orders (
    order_id        UUID PRIMARY KEY,
    stock_id        UUID REFERENCES stock(stock_id),
    user_id         UUID REFERENCES users(user_id),
    timestamp       TIMESTAMPTZ NOT NULL,
    status          order_status NOT NULL,  -- ENUM: Pending, Filed, Cancelled
    quantity        INTEGER CHECK (quantity > 0),
    price_per_stock NUMERIC(15,2) CHECK (price_per_stock > 0)
)
```

### Wallet Table
```sql
CREATE TABLE wallet (
    wallet_id      UUID PRIMARY KEY,
    user_id        UUID REFERENCES users(user_id),
    balance        NUMERIC(15,2) CHECK (balance >= 0),
    locked_balance NUMERIC(15,2) CHECK (locked_balance >= 0)
)
```

**Invariant**: `locked_balance ≤ balance`

### Watchlist Table
```sql
CREATE TABLE watchlist (
    watchlist_id   UUID NOT NULL,
    user_id        UUID REFERENCES users(user_id),
    watchlist_name VARCHAR(255) NOT NULL,
    stock_id       UUID REFERENCES stock(stock_id),
    quantity       INTEGER CHECK (quantity > 0),
    price          NUMERIC(15,2) CHECK (price > 0),
    timestamp      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (watchlist_id, stock_id)
)
```

---

## Security Mechanisms

### 1. Authentication
- **Password Storage**: bcrypt hashing (cost factor 10)
- **Token-Based**: JWT with HS256 signing
- **Token Separation**: Access (short-lived) vs Refresh (long-lived)

### 2. Authorization
- **Middleware**: JWT validation on protected routes
- **Context Injection**: UserID extracted from token → request context
- **Route Protection**: Public vs Protected route segregation

### 3. Session Invalidation
- **Redis Blacklist**: Revoked tokens stored with TTL
- **Logout**: Immediate token revocation
- **Token Expiry**: Automatic cleanup via Redis TTL

### 4. Database Security
- **Prepared Statements**: All queries use parameterized inputs (SQL injection prevention)
- **Row-Level Locking**: `FOR UPDATE` in concurrent transactions
- **Foreign Keys**: Referential integrity enforcement
- **Check Constraints**: Data validation at DB level

### 5. CORS (Future Implementation)
- Currently not implemented in shown code
- Should restrict origins in production

---

## Transaction Safety

### ACID Compliance
1. **Atomicity**: All-or-nothing via `tx.Commit()`/`tx.Rollback()`
2. **Consistency**: Check constraints + business logic validation
3. **Isolation**: `FOR UPDATE` locks during wallet operations
4. **Durability**: PostgreSQL WAL (Write-Ahead Logging)

### Race Condition Prevention
- **Wallet Updates**: Pessimistic locking (`SELECT ... FOR UPDATE`)
- **Portfolio Consistency**: Single transaction for order + wallet + portfolio
- **Idempotency**: UUID-based primary keys prevent duplicates

### Error Handling
- Database errors → Transaction rollback
- Validation failures → 400 Bad Request (no DB changes)
- Concurrent conflicts → Retry or error (depending on scenario)

---

## Frontend State Management

### Authentication State
- **Storage**: localStorage
- **Keys**: `isLoggedIn`, `access_token`, `refresh_token`, `user_id`
- **Protected Routes**: ProtectedRoute component checks auth state

### Theme Management
- **Context**: ThemeProvider (ThemeContext)
- **Persistence**: Likely localStorage (not shown in read files)

### Route Protection
```typescript
const isAuthenticated = () => {
  return localStorage.getItem("isLoggedIn") === "true";
};

const ProtectedRoute = ({ children }) => {
  return isAuthenticated() ? children : <Navigate to="/login" />;
};
```

---

## API Endpoints Summary

### Public Endpoints (No Auth Required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/auth/register | User registration |
| POST | /api/auth/login | User login |
| POST | /api/auth/refresh | Refresh access token |
| POST | /api/auth/logout | Logout user |
| GET | /api/stocks | Get all stocks |
| GET | /api/stocks/search | Search stocks by name/symbol |
| GET | /api/stocks/{stock_id} | Get stock details |
| GET | /api/stocks/{stock_id}/stats | Get stock statistics |
| GET | /api/health | Health check |

### Protected Endpoints (JWT Required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/transactions/buy | Buy stocks |
| POST | /api/transactions/sell | Sell stocks |
| GET | /api/transactions/history | Transaction history |
| GET | /api/portfolio | Get user portfolio |
| GET | /api/wallet | Get wallet info |
| GET | /api/watchlist | Get user watchlist |
| POST | /api/watchlist | Add to watchlist |
| DELETE | /api/watchlist/{watchlist_id} | Remove from watchlist |

### Admin Endpoints (Protected + Future Role Check)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/admin/stocks | Create stock |
| PUT | /api/admin/stocks/{stock_id} | Update stock |
| DELETE | /api/admin/stocks/{stock_id} | Delete stock |
| GET | /api/admin/stocks/top | Get top stocks |

---

## Error Responses

### Standard Error Format
```json
{
  "error": "error message description"
}
```

### HTTP Status Codes
- **200 OK**: Successful GET/PUT
- **201 Created**: Successful POST (resource created)
- **400 Bad Request**: Validation error, invalid input
- **401 Unauthorized**: Missing/invalid JWT
- **404 Not Found**: Resource doesn't exist
- **409 Conflict**: Duplicate resource (e.g., email exists)
- **500 Internal Server Error**: Database/server error

---

## Deployment Considerations

### Environment Variables
- `POSTGRES_USER`: Database user
- `POSTGRES_PASSWORD`: Database password
- `POSTGRES_DB`: Database name
- `POSTGRES_PORT`: Database port (default: 5432)
- `REDIS_ADDR`: Redis address (default: localhost:6379)
- `REDIS_PORT`: Redis port (default: 6379)
- `SERVER_ADDR`: API server address (default: :8080)
- `JWT_ACCESS_SECRET`: Access token signing secret
- `JWT_REFRESH_SECRET`: Refresh token signing secret

### Startup Sequence
1. PostgreSQL starts (TimescaleDB container)
2. Redis starts (Redis container)
3. Go backend initializes:
   - Connect to PostgreSQL
   - Connect to Redis
   - Initialize wallets for existing users
   - Start HTTP server
4. React frontend builds and serves (Vite dev server)

### Database Migrations
- Schema file: `backend/go/migrations/schema.sql`
- Manual execution required (no auto-migration shown)
- Tables created with `CREATE TABLE IF NOT EXISTS`

---

## Performance Optimizations

### Database
- **Indexes**: Likely on foreign keys (users.user_id, stock.stock_id)
- **Connection Pooling**: pgx pool (db.Pool)
- **TimescaleDB**: Optimized for time-series queries on portfolio/orders

### Caching
- **Redis**: Token blacklist (reduces DB hits)
- **Future**: Stock price caching, frequent query results

### Frontend
- **Code Splitting**: Vite's automatic chunking
- **Lazy Loading**: React Router with Component prop
- **Tree Shaking**: ES modules + Vite optimization

---

## Known Limitations & Future Enhancements

### Current Limitations
1. **No Role-Based Access Control**: Admin endpoints lack role verification
2. **Basic Auth Check**: Frontend auth uses simple localStorage flag
3. **No Rate Limiting**: API endpoints not protected from abuse
4. **No WebSocket**: Real-time price updates not implemented
5. **Single Portfolio**: Users have one default portfolio
6. **Stock Symbol Missing**: Database schema lacks symbol field (added in code)
7. **No Order Cancellation**: Pending orders cannot be cancelled via API
8. **Locked Balance Not Unlocked**: Buy orders lock funds but no unlock mechanism shown

### Recommended Enhancements
1. Implement role-based access (admin, user, verified user)
2. Add WebSocket for real-time stock prices
3. Implement order book matching system
4. Add market orders (limit orders currently)
5. Portfolio analytics and profit/loss tracking
6. Email verification flow
7. Two-factor authentication
8. Audit logging for all transactions
9. Backup and disaster recovery
10. API rate limiting and throttling

---

## Development Workflow

### Running Locally
1. Start infrastructure:
   ```bash
   docker-compose up -d
   ```

2. Run backend:
   ```bash
   cd backend/go
   go run main.go
   ```

3. Run frontend:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

4. Access:
   - Frontend: http://localhost:5173
   - Backend: http://localhost:8080
   - Database: localhost:5432
   - Redis: localhost:6379

---

## Conclusion

This stock trading platform implements a robust architecture with proper transaction handling, JWT-based authentication, and time-series data management. The system follows modern best practices with React frontend, Go backend, and PostgreSQL database, ensuring ACID compliance for financial transactions and secure user authentication flows.

The architecture is designed for scalability with potential enhancements including real-time updates, advanced order types, and comprehensive analytics capabilities.
