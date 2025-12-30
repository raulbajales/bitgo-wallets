#!/bin/bash

echo "🧪 Testing BitGo Request Logging Fix"
echo "===================================="

# Wait for API to be ready
echo "⏳ Waiting for API..."
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

# Test wallet creation
echo "🔧 Creating test wallet..."
response=$(curl -s -w "%{http_code}" -X POST http://localhost:8080/api/v1/wallets \
    -H 'Content-Type: application/json' \
    -d '{
        "bitgo_wallet_id": "test_wallet_debug_123",
        "label": "Debug Test Wallet",
        "coin": "btc",
        "wallet_type": "custodial",
        "threshold": 2,
        "tags": ["test", "debug", "logging"]
    }')

echo "📋 Response: $response"
echo "✅ Wallet creation test completed!"
echo ""
echo "🔍 Check the application logs for BitGo API request logging messages:"
echo "   - Look for '🚀 Capturing BitGo API request start'"
echo "   - Look for '📨 Capturing BitGo API response'"
echo "   - Look for '📋 Created BitGo request log'"
echo "   - Look for '📨 LogRequest called for'"