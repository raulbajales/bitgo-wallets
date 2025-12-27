#!/bin/bash

echo "🚀 BitGo Wallets Platform - Quick Start"
echo "======================================="

# Function to check if a port is in use
check_port() {
    if lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  Port $1 is already in use"
        return 1
    fi
    return 0
}

# Function to cleanup background processes on script exit
cleanup() {
    echo "🧹 Cleaning up..."
    pkill -f "bitgo-api" 2>/dev/null || true
    docker stop bitgo-postgres 2>/dev/null || true
    docker rm bitgo-postgres 2>/dev/null || true
}

# Set up cleanup on script exit
trap cleanup EXIT

# Check prerequisites
echo "🔍 Checking prerequisites..."
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed"; exit 1; }

# Check ports
check_port 8080 || { echo "❌ Port 8080 is required for API"; exit 1; }
check_port 5433 || { echo "❌ Port 5433 is required for Database"; exit 1; }

# Clean up any existing containers
docker stop bitgo-postgres 2>/dev/null || true
docker rm bitgo-postgres 2>/dev/null || true

# Start PostgreSQL
echo "🐘 Starting PostgreSQL..."
docker run -d \
  --name bitgo-postgres \
  -e POSTGRES_DB=bitgo_wallets \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5433:5432 \
  -v "$(pwd)/api/migrations:/docker-entrypoint-initdb.d" \
  postgres:16-alpine

# Wait for database to be ready
echo "⏳ Waiting for database..."
sleep 8
until docker exec bitgo-postgres pg_isready -U postgres > /dev/null 2>&1; do
    echo "  Database starting..."
    sleep 2
done

echo "✅ Database ready!"

# Build and start API
echo "🔧 Building and starting API..."
cd api
go build -o bitgo-api ./cmd/server/main.go

# Start API in background
DATABASE_URL='postgres://postgres:postgres@localhost:5433/bitgo_wallets?sslmode=disable' \
GIN_MODE=release \
./bitgo-api &

API_PID=$!

# Wait for API to start
echo "⏳ Waiting for API..."
sleep 3

# Test API
if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ API is ready!"
else
    echo "❌ API failed to start"
    exit 1
fi

# Start web app (if Next.js is available)
if [ -f "../web/package.json" ]; then
    echo "🌐 Starting Web App..."
    cd ../web
    if [ ! -d "node_modules" ]; then
        echo "📦 Installing dependencies..."
        npm install
    fi
    
    # Set environment variables
    export API_URL="http://localhost:8080"
    export NEXT_PUBLIC_API_URL="http://localhost:8080"
    
    npm run dev &
    WEB_PID=$!
    cd ..
    
    echo "⏳ Starting web server..."
    sleep 5
    echo "✅ Web app should be starting at http://localhost:3000"
else
    cd ..
fi

echo ""
echo "🎉 BitGo Wallets Platform Started!"
echo ""
echo "📋 Services:"
echo "   • 🗄️  Database:   localhost:5433 (postgres/postgres)"
echo "   • 🔧  API:        http://localhost:8080"
echo "   • 🌐  Web App:    http://localhost:3000"
echo ""
echo "🔑 Demo Credentials:"
echo "   • Email:     admin@bitgo.com"
echo "   • Password:  admin123"
echo ""
echo "🧪 Test Commands:"
echo "   curl http://localhost:8080/health"
echo "   curl -X POST http://localhost:8080/api/v1/auth/login \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"email\":\"admin@bitgo.com\",\"password\":\"admin123\"}'"
echo ""
echo "Press Ctrl+C to stop all services..."

# Wait for user interrupt
wait