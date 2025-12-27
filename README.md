# BitGo Wallets Platform

A full-stack custodial wallet management platform built with **Next.js** (frontend) and **Go** (backend API), supporting both Warm and Cold wallet workflows with BitGo integration.

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Git

### One-Command Setup

```bash
# Clone and start the entire stack
git clone <your-repo>
cd bitgo-wallets
./start.sh
```

**That's it!** The platform will be running at:

- 🌐 **Web App**: http://localhost:3000
- 🔧 **API**: http://localhost:8080
- 🗄️ **Database**: localhost:5433

## 🔑 Demo Credentials

### Admin Login

```
Email: admin@bitgo.com
Password: admin123
```

### API Authentication

After logging in via the web app, or directly via API:

```bash
# Login to get token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@bitgo.com","password":"admin123"}'

# Use token for API calls
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/wallets
```

## ⚙️ Environment Variables

### API Service (`api/.env`)

| Variable         | Description                  | Default                                                                     | Required |
| ---------------- | ---------------------------- | --------------------------------------------------------------------------- | -------- |
| `DATABASE_URL`   | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/bitgo_wallets?sslmode=disable` | Yes      |
| `PORT`           | API server port              | `8080`                                                                      | No       |
| `GIN_MODE`       | Gin framework mode           | `debug`                                                                     | No       |
| `ADMIN_EMAIL`    | Demo admin email             | `admin@bitgo.com`                                                           | Yes      |
| `ADMIN_PASSWORD` | Demo admin password          | `admin123`                                                                  | Yes      |

#### Future BitGo Integration

| Variable             | Description            | Default                      | Required |
| -------------------- | ---------------------- | ---------------------------- | -------- |
| `BITGO_API_URL`      | BitGo API endpoint     | `https://app.bitgo-test.com` | No       |
| `BITGO_ACCESS_TOKEN` | BitGo API access token | -                            | No       |
| `BITGO_ENVIRONMENT`  | BitGo environment      | `test`                       | No       |

### Web App (`web/.env.local`)

| Variable              | Description                    | Default                 | Required |
| --------------------- | ------------------------------ | ----------------------- | -------- |
| `API_URL`             | Internal API URL (server-side) | `http://localhost:8080` | Yes      |
| `NEXT_PUBLIC_API_URL` | Public API URL (client-side)   | `http://localhost:8080` | Yes      |
| `NODE_ENV`            | Node environment               | `development`           | No       |

### Database (`docker-compose.yml`)

| Variable            | Description       | Default         | Required |
| ------------------- | ----------------- | --------------- | -------- |
| `POSTGRES_DB`       | Database name     | `bitgo_wallets` | Yes      |
| `POSTGRES_USER`     | Database user     | `postgres`      | Yes      |
| `POSTGRES_PASSWORD` | Database password | `postgres`      | Yes      |

## 🐳 Docker Services

### Full Stack (Recommended)

```bash
# Start all services (uses the optimized startup script)
./start.sh

# Alternative: using docker-compose directly
docker-compose up

# Run in background
docker-compose up -d --build

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### Individual Services

```bash
# Start only database
docker-compose up db

# Start API + Database
docker-compose up api db

# Rebuild specific service
docker-compose up --build api
```

## 🛠️ Development Setup

### Local Development (without Docker)

#### 1. Database Setup

```bash
# Start PostgreSQL
docker run -d \
  --name bitgo-postgres \
  -e POSTGRES_DB=bitgo_wallets \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:16-alpine

# Run migrations
psql -h localhost -U postgres -d bitgo_wallets -f api/migrations/001_initial_schema.sql
```

#### 2. API Development

```bash
cd api

# Install Go dependencies
go mod download

# Create environment file
cp .env.example .env

# Install Air for hot reload (optional)
go install github.com/cosmtrek/air@latest

# Start with hot reload
air -c .air.toml

# Or start normally
go run cmd/server/main.go
```

#### 3. Web App Development

```bash
cd web

# Install dependencies
npm install
# or
pnpm install

# Create environment file
cp .env.local.example .env.local

# Start development server
npm run dev
# or
pnpm dev
```

## 📁 Project Structure

```
bitgo-wallets/
├── api/                     # Go REST API
│   ├── cmd/server/         # Application entry point
│   ├── internal/           # Internal packages
│   │   ├── api/           # HTTP handlers & routes
│   │   ├── config/        # Configuration management
│   │   ├── database/      # Database connection
│   │   ├── models/        # Data models
│   │   └── repository/    # Data access layer
│   ├── migrations/        # SQL migration files
│   ├── .env.example       # Environment template
│   ├── .air.toml          # Hot reload config
│   └── Dockerfile         # API container
├── web/                    # Next.js Frontend
│   ├── src/app/           # Next.js 13+ App Router
│   ├── .env.local.example # Environment template
│   └── Dockerfile.dev     # Web container
├── docker-compose.yml     # Full stack orchestration
└── README.md             # This file
```

## 🔄 API Endpoints

### Authentication

- `POST /api/v1/auth/login` - Login with email/password

### Health Check

- `GET /health` - Service health status

### Wallets (Protected)

- `GET /api/v1/wallets` - List wallets
- `POST /api/v1/wallets` - Create wallet
- `GET /api/v1/wallets/:id` - Get wallet details
- `PUT /api/v1/wallets/:id` - Update wallet
- `DELETE /api/v1/wallets/:id` - Delete wallet

### Transfers (Protected)

- `GET /api/v1/wallets/:id/transfers` - List transfers for wallet
- `POST /api/v1/wallets/:id/transfers` - Create transfer request
- `GET /api/v1/transfers/:id` - Get transfer details
- `PUT /api/v1/transfers/:id/status` - Update transfer status

## 📊 Database Schema

### Core Tables

- **users** - User accounts with role-based access
- **organizations** - Multi-tenant organization support
- **wallets** - BitGo wallet mappings (warm/cold types)
- **wallet_memberships** - User-wallet associations
- **transfer_requests** - Transfer lifecycle tracking
- **audit_logs** - Comprehensive audit trail

### Wallet Types

- **Warm Wallets**: Online wallets for fast transactions
- **Cold Wallets**: Offline wallets for secure long-term storage

### Transfer Status Flow

```
draft → submitted → pending_approval → approved → completed
                                    ↘ rejected
                                    ↘ failed
                                    ↘ cancelled
```

## 🔒 Security Features

- **Role-based Access Control**: End User, Operator, Approver, Admin roles
- **Audit Logging**: Complete audit trail for all actions
- **Multi-signature Support**: Configurable signature thresholds
- **Transfer Approvals**: Configurable approval workflows
- **Environment Isolation**: Separate dev/test/prod configurations

## 🧪 Testing

### API Testing

```bash
# Health check
curl http://localhost:8080/health

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@bitgo.com","password":"admin123"}'

# Create wallet (with auth token)
curl -X POST http://localhost:8080/api/v1/wallets \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "bitgo_wallet_id": "test_wallet_123",
    "label": "Test Wallet",
    "coin": "tbtc4",
    "wallet_type": "warm"
  }'
```

### Database Access

```bash
# Connect to database
docker exec -it bitgo-wallets-db-1 psql -U postgres -d bitgo_wallets

# Common queries
SELECT * FROM wallets;
SELECT * FROM transfer_requests;
SELECT * FROM users;
```

## 🚧 Current Implementation Status

### ✅ Completed (Milestone 2)

- PostgreSQL database with complete schema
- Go API with Gin framework
- Next.js web application
- Docker containerization
- Basic authentication
- Wallet and transfer CRUD operations
- Audit logging infrastructure

### 🔄 Next Steps (Milestone 3)

- BitGo API integration
- Real wallet discovery
- Transfer execution
- Webhook handling
- Status synchronization

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📝 License

[Your License Here]

## 🆘 Troubleshooting

### Common Issues

**Docker Permission Denied (Linux)**

```bash
# Option 1: Run with sudo (quick fix)
sudo docker-compose up --build

# Option 2: Add user to docker group (permanent fix)
sudo usermod -aG docker $USER
# Then logout and login again, or run:
newgrp docker

# Option 3: Fix docker socket permissions
sudo chmod 666 /var/run/docker.sock
```

**Database Connection Failed**

```bash
# Check if PostgreSQL is running
docker-compose ps

# View database logs
docker-compose logs db

# Reset database
docker-compose down -v
docker-compose up --build
```

**API Not Starting**

```bash
# Check API logs
docker-compose logs api

# Rebuild API container
docker-compose up --build api

# Check environment variables
docker-compose exec api env
```

**Web App Not Loading**

```bash
# Check web logs
docker-compose logs web

# Rebuild web container
docker-compose up --build web

# Check if API is accessible
curl http://localhost:8080/health
```

### Reset Everything

```bash
# Complete reset (removes all data)
docker-compose down -v
docker system prune -f
docker-compose up --build
```
