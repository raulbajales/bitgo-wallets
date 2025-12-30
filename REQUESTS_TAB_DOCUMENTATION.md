# BitGo Wallets Platform - New "Requests" Tab Documentation

## Overview

Successfully implemented a new **"Requests"** tab in the Dashboard that provides a developer console for debugging BitGo API interactions.

## Features Implemented

### 🔍 API Request Logging

- **Real-time capture** of all API requests made through the application
- **Automatic logging** of request details, responses, and errors
- **Request/Response timing** with duration tracking

### 🛡️ Security Features

- **Token obscuration**: Auth tokens are automatically masked in the console (shows first 6 and last 4 characters)
- **Secure display**: Only non-sensitive headers and data are fully visible

### 📋 Console Display Features

- **Terminal-style interface** with green text on dark background
- **Chronological display** (most recent requests at the top)
- **CURL command generation** for easy reproduction of requests
- **Pretty-printed JSON** responses with syntax highlighting
- **Clear logs** functionality to reset the console

### 🎯 Request Information Displayed

For each API request, the console shows:

1. **Timestamp**: Precise time of request (HH:MM:SS format)
2. **HTTP Method**: GET, POST, PUT, DELETE
3. **Request URL**: Full endpoint URL
4. **Headers**: All headers with obscured auth token
5. **Request Body**: JSON payload (if applicable)
6. **Response Status**: HTTP status code with color coding
7. **Response Body**: Pretty-formatted JSON response
8. **Duration**: Request completion time in milliseconds
9. **Errors**: Detailed error messages if request fails

### 🎨 Visual Features

- **Color-coded status indicators**:
  - 🟢 Green: Successful requests (2xx status)
  - 🟡 Yellow: Client errors (4xx status)
  - 🔴 Red: Server errors/network failures
- **Scrollable console** with fixed height for easy navigation
- **Responsive layout** that works on different screen sizes
- **Clear visual separation** between different requests

### 🔧 Technical Implementation

- **Integrated with existing API client** (`/web/src/lib/api.ts`)
- **Non-intrusive logging** that doesn't affect application performance
- **Type-safe implementation** with full TypeScript support
- **Memory efficient** with manual clear functionality

## Usage Instructions

1. **Navigate to Requests Tab**: Click on the "Requests" tab in the main navigation
2. **Generate API Calls**: Interact with wallets, transfers, or any other features that make API calls
3. **Monitor Requests**: See real-time requests appearing in the console
4. **Debug Issues**: Use the CURL commands to reproduce requests manually
5. **Clear History**: Use the "Clear Logs" button to reset the console

## Developer Benefits

### 🐛 Debugging

- **Immediate visibility** into all API interactions
- **CURL generation** for manual testing and reproduction
- **Error details** for troubleshooting failed requests
- **Performance monitoring** with request duration tracking

### 📖 Learning & Documentation

- **Understand data flow** between frontend and BitGo API
- **Copy-paste CURL commands** for API documentation
- **Real-time examples** of proper request formatting
- **Response structure inspection** for integration development

### 🔒 Security Verification

- **Token handling verification** (tokens are properly obscured)
- **Request validation** ensuring proper headers and payloads
- **Response analysis** for security-related responses

## Example Console Output

```
[17:45:23] GET Request                                    [200] 125ms
CURL:
curl -X GET "http://localhost:8080/api/v1/wallets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJ0eX...3kQ2"

Response:
{
  "wallets": [
    {
      "id": "wallet_123",
      "name": "Main BTC Wallet",
      "coin": "btc",
      "balance": "1.50000000",
      ...
    }
  ]
}
```

This implementation provides developers with comprehensive visibility into API interactions, making debugging and development much more efficient while maintaining security best practices.
