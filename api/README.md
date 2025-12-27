# BitGo Wallets API

This is the Go REST API for the BitGo Wallets platform.

## Features

- **Database**: PostgreSQL with raw SQL queries
- **Framework**: Gin HTTP framework
- **Authentication**: Basic token-based auth (demo implementation)
- **Models**: Users, Organizations, Wallets, Transfer Requests, Audit Logs
- **Hot Reload**: Air for development

## Quick Start

### Using Docker Compose (Recommended)

From the project root:

```bash
docker-compose up --build
```

This will start:

- PostgreSQL database on port 5432
- Go API on port 8080 with hot reload
- Next.js web app on port 3000

### Manual Setup

1. **Install Dependencies:**

   ```bash
   go mod download
   ```

2. **Setup Database:**
   ````bash
   # Start PostgreSQL
   docker run -d \
     --name bitgo-postgres \
     -e POSTGRES_DB=bitgo_wallets \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=postgres \
     -p 5432:5432 \
     postgres:16-alpine\n
   # Run migrations\n   psql -h localhost -U postgres -d bitgo_wallets -f migrations/001_initial_schema.sql\n   ```\n\n3. **Start Server:**\n   ```bash\n   go run cmd/server/main.go\n   ```\n\n## API Endpoints\n\n### Authentication\n- `POST /api/v1/auth/login` - Login with email/password\n\n### Health Check\n- `GET /health` - Service health status\n\n### Wallets (Protected)\n- `GET /api/v1/wallets` - List wallets\n- `POST /api/v1/wallets` - Create wallet\n- `GET /api/v1/wallets/:id` - Get wallet\n- `PUT /api/v1/wallets/:id` - Update wallet\n- `DELETE /api/v1/wallets/:id` - Delete wallet\n\n### Transfers (Protected)\n- `GET /api/v1/wallets/:id/transfers` - List transfers for wallet\n- `POST /api/v1/wallets/:id/transfers` - Create transfer\n- `GET /api/v1/transfers/:id` - Get transfer\n- `PUT /api/v1/transfers/:id` - Update transfer\n- `PUT /api/v1/transfers/:id/status` - Update transfer status\n\n## Database Schema\n\n### Tables\n- `users` - User accounts and roles\n- `organizations` - Multi-tenant organizations\n- `wallets` - BitGo wallet mappings (warm/cold)\n- `wallet_memberships` - User-wallet associations\n- `transfer_requests` - Transfer requests with status tracking\n- `audit_logs` - Comprehensive audit trail\n\n### Wallet Types\n- **Warm**: Online wallets for fast transactions\n- **Cold**: Offline wallets for secure storage\n\n### Transfer Status Flow\n1. `draft` → `submitted` → `pending_approval` → `approved` → `completed`\n2. Alternative: `rejected`, `failed`, `cancelled`\n\n## Authentication (Demo)\n\n**Login Credentials:**\n- Email: `admin@bitgo.com`\n- Password: `admin123`\n\n**Usage:**\n```bash\n# Login\ncurl -X POST http://localhost:8080/api/v1/auth/login \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"email\":\"admin@bitgo.com\",\"password\":\"admin123\"}'\n\n# Use token in subsequent requests\ncurl -H \"Authorization: Bearer YOUR_TOKEN\" \\\n  http://localhost:8080/api/v1/wallets\n```\n\n## Development\n\n### Project Structure\n```\napi/\n├── cmd/server/          # Application entry point\n├── internal/\n│   ├── api/             # HTTP handlers and routes\n│   ├── config/          # Configuration\n│   ├── database/        # Database connection\n│   ├── models/          # Data models\n│   └── repository/      # Data access layer\n├── migrations/          # SQL migration files\n└── .air.toml           # Hot reload configuration\n```\n\n### Database Migrations\n\nMigrations are plain SQL files in `/migrations/`. They run automatically when the database container starts.\n\n### Environment Variables\n\n- `DATABASE_URL` - PostgreSQL connection string\n- `PORT` - API server port (default: 8080)\n- `GIN_MODE` - Gin mode (debug/release)\n- `ADMIN_EMAIL` - Demo admin email\n- `ADMIN_PASSWORD` - Demo admin password\n\n## Next Steps (Milestone 3)\n\n- Integrate with BitGo API\n- Implement real JWT authentication\n- Add webhook handling\n- Implement transfer status polling\n- Add comprehensive error handling\n- Add logging and correlation IDs
   ````
