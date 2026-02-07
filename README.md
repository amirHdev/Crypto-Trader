# Crypto Trader

**KuCoin Scheduled Buy & Trailing Sell Bot**

Desktop GUI • Telegram notifications • Persistent scheduled orders

A lightweight KuCoin spot trading bot with a simple desktop interface that allows you to:

- Schedule market buys for any supported trading pair (e.g. BTC-USDT)
- Automatically sell when price reaches a user-defined profit target
- Control position size using a percentage of available USDT balance
- Receive Telegram notifications for every important event
- View and manage all scheduled & past orders

**Current status (February 2026)** — **early development / proof-of-concept**  
**NOT READY FOR REAL MONEY — use sandbox or very small amounts only**

## ⚠️ Critical Warnings

Before you continue — please read this carefully:

- The current version **does not correctly specify amount/size** in market buy & sell orders  
  → KuCoin will most likely **reject the orders** or fill only dust amounts
- No real fill-size tracking → sell logic may attempt to sell wrong quantity
- No balance/position check before selling → risk of attempting to sell assets you don't hold
- No stop-loss / trailing-stop / max holding time protection
- No duplicate order prevention after app restart
- Trading fees are **not** taken into account in profit calculation

**Do NOT use real funds until at least the top 4 fixes listed in the roadmap below are implemented and tested.**

## Features

- Cross-platform desktop GUI (Fyne)
- Schedule future market buys with % of wallet
- Automatic market sell on fixed % profit target
- Telegram notifications (schedule, buy executed, sell triggered, errors)
- Persistent order storage using Buntdb
- Public WebSocket price monitoring with auto-reconnect
- Structured logging (zerolog)

## Quick Start

1. **Recommended: use KuCoin sandbox**  
   https://sandbox.kucoin.com  
   Create separate sandbox API keys there.

2. Copy `.env.example` → `.env` and fill in your credentials:

   ```env
   KUCOIN_API_KEY=your_key_here
   KUCOIN_API_SECRET=your_secret_here
   KUCOIN_API_PASSPHRASE=your_passphrase_here
   TELEGRAM_BOT_TOKEN=your_bot_token_here
   TELEGRAM_CHAT_IDS=123456789,987654321 # comma separated
   DATA_DIR=./data
   ```

- For sandbox testing, also set the sandbox base URL if your code supports it (check `/internal/config` or similar — many KuCoin SDKs have a `sandbox: true` flag).
- Create a Telegram bot via @BotFather to get the token, and get your chat ID(s) by messaging @userinfobot or similar.

3. Run the bot:

   ```bash
   # Option 1 — run directly (recommended during dev)
   go run ./cmd/trader

   # Option 2 — build standalone binary
   go build -o crypto-trader ./cmd/trader
   ./crypto-trader               # Linux/macOS
   crypto-trader.exe             # Windows
   ```

## Development

```bash
# Download / tidy dependencies
go mod tidy

# Run with live reload (optional — requires air)
go install github.com/air-verse/air@latest
air

# Build for current OS
go build -o crypto-trader ./cmd/trader

# Cross-compile example (Windows 64-bit)
GOOS=windows GOARCH=amd64 go build -o crypto-trader.exe ./cmd/trader
```

## Roadmap – Critical Fixes Needed (2026)

1. Fix market buy & sell orders — properly set **funds** (for buy in quote currency e.g. USDT amount) or **size** (for sell in base currency quantity) per KuCoin API docs
2. Track real filled quantity after buy (via `/api/v1/fills` endpoint, `/api/v1/orders/{order-id}`, or repeated balance queries)
3. Check available balance/holdings before placing sell order (via `/api/v1/accounts` or spot holdings endpoint)
4. Prevent duplicate scheduling on app restart / restore (e.g. unique clientOrderId + check against open orders on startup)
5. Add basic stop-loss or maximum holding time
6. Improve Telegram MarkdownV2 message escaping
7. Validate trading pair symbol before saving order (fetch from `/api/v1/symbols` or cache)
8. Add rate limiting / retry logic for REST API calls (KuCoin enforces strict limits)
9. (Nice to have) Private WebSocket for own trades & balance updates

## License

GNU General Public License v3.0

You are **strongly discouraged** from using this software with real money in its current form.

Contributions are welcome — especially pull requests that fix any of the critical issues listed above.
