# BillionCore - High-Performance Real-Time BIN Decision Engine 🚀

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Language](https://img.shields.io/badge/language-Go-00ADD8.svg)
![Speed](https://img.shields.io/badge/latency-16μs-brightgreen.svg)

**BillionCore** is a lightning-fast, zero-dependency BIN (Bank Identification Number) decision engine built in pure Go. Engineered for ultra-low latency fintech environments, it enables real-time transaction routing, fraud prevention, and payment optimization with sub-millisecond precision.

---

## ⚡ Performance Reality

While traditional tech stacks (PHP, Python, Node.js) struggle with overhead and garbage collection pauses, BillionCore is optimized for bare-metal performance on a single vCPU.

| Metric | BillionCore Performance |
| :--- | :--- |
| **Engine Internal Latency** | **16.3 microseconds (µs)** |
| **Throughput (Internal)** | **5,500+ Requests/sec** |
| **Throughput (Public Proxy)** | **3,250+ Requests/sec** (Limited by Network/Nginx) |
| **Memory Footprint** | **~12MB** (under heavy load) |
| **Reliability** | **0% Errors** under 1,000 concurrent connections |

---

## 🏗️ Technical Highlights

- **Zero-Dependency Architecture**: Built using only the Go Standard Library. No external bloat, no security vulnerabilities from third-party packages, and maximum stability.
- **In-Memory Rule Evaluation**: Optimized for high-concurrency environments using Go's native data structures, ensuring O(1) or O(log N) lookup complexity.
- **Concurrent-Safe Processing**: Designed to handle asynchronous surges in transaction volume without performance degradation.
- **Transparent Benchmarking**: Direct internal timing headers are exposed for real-world auditing through the `X-Internal-Nanoseconds` header.

---

## 📂 Project Structure

```text
.
├── cmd/                # Entry point for the API
├── internal/
│   ├── api/            # High-performance HTTP handlers
│   ├── store/          # Efficient rule loading and memory management
│   ├── models/         # Lightweight data structures
│   └── logger/         # Low-overhead structured logging
├── main.go             # Application bootstrap
└── Dockerfile          # Reproducible production builds
```

---

## 🚀 Quick Start

### 1. Requirements
- Go 1.25.6 or later
- Docker (optional)

### 2. Local Installation
```bash
git clone git@github.com:Nikita95727/BillionCore.git
cd BillionCore
go build -o bin-engine main.go
./bin-engine
```

### 3. Docker Deployment
```bash
docker build -t billioncore-engine .
docker run -p 8083:8083 billioncore-engine
```

---

## 🔌 API Reference

The BillionCore Engine exposes a minimalist REST API designed for high-throughput consumption.

### 1. BIN Lookup
**Endpoint**: `GET /api/bin/lookup`

**Headers**:
- `X-API-Key`: Your secure API authentication key.

**Query Parameters**:
- `bin` (required): The 6 or 8-digit BIN number to verify.
- `country` (required): ISO-2 country code (e.g., `US`, `GB`, `ES`).

#### Sample Request
```bash
curl -X GET "http://localhost:8083/api/bin/lookup?bin=457112&country=US" \
     -H "X-API-Key: your_api_key_here"
```

#### Sample Response
```json
{
  "success": true,
  "data": {
    "bin": "457112",
    "country": "US",
    "action": "ENABLE",
    "trial_price": 1.00,
    "trial_period": 3,
    "rebill_price": 49.99,
    "rebill_period": 30,
    "x_sell_status": "ENABLE",
    "bin_performance": {
      "gross_profit": 85.5,
      "lead_u": 1.25,
      "first_rebill": 42.1,
      "rebill": 15.6,
      "tc40_safe": 98.2,
      "cb": 0.05,
      "refund": 1.2
    }
  },
  "timestamp": "2026-04-19T13:15:00Z"
}
```

### 2. Monitoring Performance
Every response includes custom headers that expose the internal engine processing time, independent of network latency:
- `X-Internal-Time`: Human-readable latency (e.g., `16.3µs`).
- `X-Internal-Nanoseconds`: Raw nanosecond timing for high-precision auditing.

---

## 🔬 Use Cases

- **Real-Time Payment Routing**: Decide the best payment provider based on BIN data in under 20 microseconds.
- **Fraud Prevention**: Instantly block or flag high-risk BIN ranges before the transaction hits the processing layer.
- **Dynamic Pricing**: Adjust trial periods and rebill logic based on card issuer data on-the-fly.

---

## 📜 Grant & Investment Application

BillionCore represents a generational leap in payment processing infrastructure. By reducing the decision-making window to microseconds, we enable enterprises to save millions in transaction fees and prevent fraudulent activities at the "edge" of their network.

**Built with pride by Nikita Tretynko.**

---

## ⚖️ License
Distributed under the MIT License. See `LICENSE` for more information.
