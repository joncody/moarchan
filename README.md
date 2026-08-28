# moarchan

[![Rust](https://img.shields.io/badge/Rust-2024_Edition-dea584?style=flat&logo=rust&logoColor=white)](https://www.rust-lang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-4169E1?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![JavaScript](https://img.shields.io/badge/JavaScript-Vanilla%20ES6+-F7DF1E?style=flat&logo=javascript&logoColor=black)](https://developer.mozilla.org/en-US/docs/Web/JavaScript)
[![Protocol](https://img.shields.io/badge/Protocol-HTTP%2F3%20%7C%20QUIC%20%7C%20SSE-555555?style=flat)]()
[![Gateway](https://img.shields.io/badge/Gateway-Caddy%202-00ADD8?style=flat&logo=caddy&logoColor=white)](https://caddyserver.com/)
[![Docker Compose](https://img.shields.io/badge/Docker-Compose%20Ready-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A high-performance, real-time 4chan clone built from scratch using **Rust (2024 Edition)**, **Axum**, **Tokio**, **HTTP/3 over QUIC**, **Server-Sent Events (SSE)**, **PostgreSQL (`LISTEN / NOTIFY`)**, and **Vanilla JavaScript** (strictly adhering to Douglas Crockford's coding standards).

---

## 🚀 Key Features

* **HTTP/3 & QUIC Transport:** Ultra-low-latency 0-RTT UDP transport eliminating Head-of-Line (HoL) blocking across media loading and live event streams, with automated `Alt-Svc` header discovery and Encrypted Client Hello (ECH) compatibility.
* **Distributed Real-Time SSE Hub:** Horizontally scalable pub/sub powered by PostgreSQL `LISTEN / NOTIFY` featuring `instance_id` origin-tagging to prevent local duplicate event fanout, plus payload hydration fallback for large frames (>7.5 KB).
* **Stream Backpressure & Lag Recovery:** Resilient Tokio broadcast handling that catches buffer lag to dispatch `sync-required` events, automatically triggering seamless client-side rehydration.
* **Bidirectional Live Backlinks:** Automatic real-time DOM injection of `>>hash` quote backlinks across both OPs and replies upon receiving live SSE reply events.
* **Declarative AEAD Cookie Sessions:** Cryptographically sealed client sessions utilizing `axum-extra`'s `PrivateCookieJar` with 512-bit SHA-512 master key expansion and zero database lookup overhead.
* **Finite Board Capacity & Auto-Pruning:** Strictly enforced board limits (100 threads per board) with automated asynchronous cascade deletion of orphaned media files ($O(1)$ storage ceiling).
* **Indexed Subquery Query Planning:** Board indexing using derived subquery limits (`LIMIT 100`) to eliminate full-table scan bottlenecks across active boards.
* **Chronological Bumping & Bump Limit:** Real-time thread bumping with `bumped_at` timestamps, chronological reply ordering, `sage` bypass, and an automated 300-reply bump limit.
* **Classic & Secure Tripcode Engine:** Native parser for traditional (`#password`) and salted secure (`##password`) tripcodes with dedicated styling.
* **Alpha-Safe High-Fidelity Thumbnails:** Fast integer downsampling using the `image` crate with strict dimension bomb defenses (10000x10000px validation), magic-byte format verification, EXIF stripping, and RGB8 flattening for alpha transparency safety.
* **Zero-I/O Template Engine:** In-memory pre-cached MiniJinja template rendering compiled directly into RAM on startup for sub-millisecond page and item assembly.
* **IPv6 Subnet-Aware & Proxy-Trusted Rate Limiting:** Token-bucket IP rate limiter (2 req/sec, burst 10) with `/64` CIDR bitmasking and RFC1918 / ULA / loopback reverse-proxy IP extraction.
* **Universal Double-Submit CSRF Protection:** Timing-attack resistant verification (`subtle::ConstantTimeEq`) enforced across all state-mutating HTTP methods (`POST`, `PUT`, `DELETE`, `PATCH`).
* **Pure Crockfordian JavaScript:** Modular client-side SPA runtime written with zero usage of `this`, `class`, `var`, `new` (in application code), or `void` operators.
* **Componentized Frontend Architecture:** Decomposed into dedicated ES modules (`post-renderer`, `tag-hover`, `reply-box`, `post-actions`, `post-form`) with explicit lifecycle teardowns to prevent memory leaks.
* **In-Place Image Expansion:** Clickable thumbnail expansion within feed and thread views, with clean DOM removal of outer reply containers on deletion.
* **HTML5 History API Routing:** Clean URLs (`/g`, `/g/thread/a1b2c3d4e`, static views) with deep-linking support and history navigation.
* **Graceful Shutdown & Draining:** Integrated `SIGINT`/`SIGTERM` signal listening with active connection draining.

---

> **⚠️ Note on User Accounts / Authentication:**  
> The login and registration system (`/auth`, `src/routes/auth.rs`) is included strictly as a **functional demonstration** of the session management, private cookie encryption, and bcrypt capabilities. True to traditional imageboard culture, all board browsing, thread creation, and replying remain completely open, anonymous, and account-free by default.

---

## 🛠️ Tech Stack

* **Backend:** Rust (2024 Edition)
* **Web Framework:** [Axum 0.8](https://github.com/tokio-rs/axum) / [Axum-Extra 0.12](https://docs.rs/axum-extra) / [Tower](https://github.com/tower-rs/tower) / [Hyper 1.0](https://hyper.rs/)
* **Async Runtime:** [Tokio 1.43](https://tokio.rs/)
* **Database Driver:** [SQLx (PostgreSQL 16+)](https://github.com/launchbadge/sqlx)
* **Gateway & Edge:** [Caddy 2](https://caddyserver.com/) (Automated TLS, HTTP/3 QUIC & ECH)
* **Template Engine:** [MiniJinja](https://github.com/mitsuhiko/minijinja) (In-Memory Pre-cached)
* **Image Processing:** `image` crate (Fast integer downsampling, EXIF stripping & magic-byte sniffing)
* **Frontend:** Vanilla JavaScript (ES6 Modules, Crockfordian), HTML5, CSS3
* **Protocols:** HTTP/3 over QUIC (UDP 443) / HTTP/2 / Server-Sent Events (SSE)
* **Sessions:** AES-256-GCM Encrypted `PrivateCookieJar` (`axum-extra`)

---

## 📋 Prerequisites

* **Docker & Docker Compose** (Recommended)
* *Or for local bare-metal development:* **Rust 1.85+** and **PostgreSQL 16+**

---

## 🏁 Quick Start (Production & Local Docker)

The fastest and most reliable way to run MoarChan with full HTTP/3 (QUIC) and automated TLS is using Docker Compose:

### 1. Clone the Repository
```bash
git clone https://github.com/joncody/moarchan.git
cd moarchan
```

### 2. Configure Environment Variables
Create a `.env` file in the root directory:
```ini
DOMAIN=localhost
PORT=9001
POSTGRES_HOST=db
POSTGRES_PORT=5432
POSTGRES_USER=moarchan
POSTGRES_PASSWORD=moarchan
POSTGRES_DB=moarchan
POSTGRES_SSLMODE=disable
SESSION_HASH_KEY=12345678901234567890123456789012
SESSION_BLOCK_KEY=abcdefghijklmnopqrstuvwx12345678
UPLOAD_PATH=./static/images/uploads
UPLOAD_URL_PREFIX=/static/images/uploads
VIEWS_PATH=./static/views
```

> **For Production Domains:** Change `DOMAIN=localhost` to `DOMAIN=yourdomain.com`. Caddy will automatically provision trusted Let's Encrypt / ZeroSSL certificates and configure HTTP/3 over QUIC on UDP port 443.

### 3. Build and Start
```bash
docker compose up --build -d
```

Navigate to `https://localhost` (or `https://yourdomain.com`) in your browser.

---

## 🧪 Testing the Deployment

### 1. Verify HTTP/3 (QUIC over UDP)
Test the HTTP/3 UDP handshake using a containerized HTTP/3 client:
```bash
docker run --rm --net=host ymuski/curl-http3 curl -k --http3-only -I https://localhost
```

### 2. Verify Rate Limiting (Burst Defense)
Send 15 rapid POST requests to observe the token bucket trigger `HTTP 429`:
```bash
for i in {1..15}; do
  curl -k -s -o /dev/null -w "Request $i: HTTP %{http_code}\n" -X POST https://localhost/api/threads
done
```

### 3. Verify CSRF Protection
Ensure unauthenticated mutating requests are rejected:
```bash
curl -k -X POST https://localhost/api/threads \
  -H "Content-Type: application/json" \
  -d '{"topic":"g","comment":"Unauthorized"}'
# Output: {"error":"CSRF token validation failed","status":403}
```

---

## ⚙️ Environment Configuration Reference

| Environment Variable | Default Value | Description |
| :--- | :--- | :--- |
| `DOMAIN` | `localhost` | Domain name for automated TLS and HTTP/3 gateway |
| `PORT` | `9001` | Internal server HTTP port |
| `POSTGRES_HOST` | `db` | PostgreSQL host address |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `moarchan` | PostgreSQL username |
| `POSTGRES_PASSWORD` | `moarchan` | PostgreSQL password |
| `POSTGRES_DB` | `moarchan` | Database name |
| `POSTGRES_SSLMODE` | `disable` | SSL mode (`disable`, `require`, `verify-full`) |
| `SESSION_HASH_KEY` | *(32 bytes)* | Secret key for deriving session master key |
| `SESSION_BLOCK_KEY` | *(32 bytes)* | Secret key for deriving session master key |
| `UPLOAD_PATH` | `./static/images/uploads` | Local filesystem base path for media storage |
| `UPLOAD_URL_PREFIX` | `/static/images/uploads` | Public URL prefix for uploaded media assets |
| `VIEWS_PATH` | `./static/views` | Directory path containing HTML templates |

---

## 🏗️ Project Architecture

```
.
├── Cargo.toml            # Project dependencies & build manifest
├── Caddyfile             # Gateway config (HTTP/3 QUIC, TLS, proxying)
├── Dockerfile            # Multi-stage Rust build & runtime container
├── docker-compose.yml    # Multi-container orchestration (App, DB, Gateway)
├── .env                  # Local environment configuration (git-ignored)
├── src/
│   ├── main.rs           # Application bootstrap, graceful shutdown & server launch
│   ├── config.rs         # Environment variable configuration loader
│   ├── state.rs          # Thread-safe global AppState & cookie key container
│   ├── error.rs          # Unified error handling & HTTP response conversion
│   ├── db/
│   │   ├── mod.rs        # DB module entrypoint
│   │   ├── migrations.rs # Versioned transactional schema migrations (001-006)
│   │   └── queries.rs    # Domain SQL queries & indexed aggregate builders
│   ├── middleware/
│   │   ├── mod.rs        # Middleware module entrypoint
│   │   ├── csrf.rs       # Double-submit cookie CSRF middleware (timing-attack resistant)
│   │   ├── rate_limit.rs # IPv6 /64 subnet & proxy-aware token-bucket rate limiter
│   │   └── security.rs   # Alt-Svc advertisement, CSP & defensive headers
│   ├── models/
│   │   ├── mod.rs        # Models module entrypoint
│   │   ├── auth.rs       # User authentication & session models
│   │   ├── post.rs       # Thread, Reply & File models
│   │   └── sse.rs        # Event envelope models
│   ├── routes/
│   │   ├── mod.rs        # Master Axum router builder
│   │   ├── auth.rs       # Login, registration & session handlers (PrivateCookieJar)
│   │   ├── pages.rs      # HTML base shell & SPA dynamic render handlers
│   │   └── api/
│   │       ├── mod.rs    # API subrouter
│   │       ├── threads.rs# Thread creation & auto-pruning endpoint
│   │       ├── replies.rs# Reply creation, bump limit & backlink update endpoint
│   │       ├── delete.rs # Post/file deletion endpoint
│   │       └── stream.rs # Real-time SSE stream & lag recovery (sync-required)
│   ├── services/
│   │   ├── mod.rs        # Services module entrypoint
│   │   ├── auth.rs       # Bcrypt password verification & hashing
│   │   ├── image.rs      # Thumbnailing, magic-byte checking, RGB8 alpha safety
│   │   ├── sanitizer.rs  # HTML escaping, tripcode engine & quote parsing
│   │   └── sse.rs        # Distributed Postgres LISTEN/NOTIFY SSE hub (origin-tagged)
│   └── storage/
│       ├── mod.rs        # Pluggable StorageBackend trait
│       └── local.rs      # Concurrent local filesystem storage with path sanitization
└── static/
    ├── css/              # Reset, post, thread, reply & screen stylesheet rules
    ├── images/           # Application graphics & upload directory
    │   └── uploads/      # Image uploads (git-ignored)
    ├── js/
    │   ├── frame.js      # Crockfordian SPA runtime (History API + SSE)
    │   ├── dom.js        # Lightweight DOM manipulation library
    │   ├── components/   # Modular UI components
    │   │   ├── topics-map.js     # Board slugs & descriptions map
    │   │   ├── post-renderer.js  # JSON-to-DOM HTML builder
    │   │   ├── tag-hover.js      # Quote preview tooltips & jump links
    │   │   ├── reply-box.js      # Draggable Quick Reply modal
    │   │   ├── post-actions.js   # Collapse, hide, & in-place image expansion
    │   │   └── post-form.js      # Form submissions & validation
    │   └── controllers/
    │       ├── auth.js   # Auth controller (Demo)
    │       ├── main.js   # Homepage controller
    │       └── service.js# Imageboard thread/reply orchestrator & real-time backlinks
    └── views/            # MiniJinja HTML templates
```

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.
