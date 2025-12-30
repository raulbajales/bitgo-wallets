#!/bin/bash

echo "🧪 Testing BitGo Request Logging with Authentication"
echo "=================================================="

# Wait for API to be ready
echo "⏳ Waiting for API to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        echo "✅ API is ready!"
        break
    fi
    sleep 1
    if [ $i -eq 30 ]; then
        echo "❌ API failed to start after 30 seconds"
        exit 1
    fi
done

# Login to get authentication token
echo "🔐 Logging in..."
login_response=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d '{
        "email": "admin@bitgo.com",
        "password": "admin123"
    }')

echo "📝 Login response: $login_response"

# Extract token (assuming it's in the response as "token")
token=$(echo "$login_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$token" ]; then
    echo "❌ Failed to get authentication token"
    exit 1
fi

echo "🎟️  Got token: ${token:0:20}..."

# Create wallet with authentication
echo "🔧 Creating wallet with authentication..."
wallet_response=$(curl -s -w "%{http_code}" -X POST http://localhost:8080/api/v1/wallets \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $token" \
    -d '{
        "bitgo_wallet_id": "test_auth_wallet_123",
        "label": "Authenticated Test Wallet",
        "coin": "btc",
        "wallet_type": "custodial",
        "threshold": 2,
        "tags": ["test", "auth", "debug"]
    }')

echo "📋 Wallet creation response: $wallet_response"
echo ""
echo "✅ Test completed! Check:"
echo "   1. The server logs for BitGo debug messages"
echo "   2. The Requests tab in the web UI for BitGo API calls"