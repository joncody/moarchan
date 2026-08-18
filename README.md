# moarchan

[![Rust](https://img.shields.io/badge/Rust-2024_Edition-dea584?style=flat&logo=rust&logoColor=white)](https://www.rust-lang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-4169E1?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![JavaScript](https://img.shields.io/badge/JavaScript-Vanilla%20ES6+-F7DF1E?style=flat&logo=javascript&logoColor=black)](https://developer.mozilla.org/en-US/docs/Web/JavaScript)
[![Protocol: HTTP/2 & SSE](https://img.shields.io/badge/Protocol-HTTP%2F2%20%7C%20SSE-555555?style=flat)]()
[![Docker Compose](https://img.shields.io/badge/Docker-Compose%20Ready-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A high-performance, real-time 4chan clone built from scratch using **Rust (2024 Edition)**, **Axum**, **Tokio**, **HTTP/2**, **Server-Sent Events (SSE)**, **PostgreSQL (`LISTEN / NOTIFY`)**, and **Vanilla JavaScript** (strictly following Douglas Crockford's coding standards).

---

## 🚀 Key Features

* **HTTP/2 & Real-Time SSE Bus:** Multiplexed Server-Sent Events distributed horizontally across instances using PostgreSQL `LISTEN / NOTIFY` with compact JSON descriptors and database hydration fallback.
* **Stream Backpressure & Lag Resilience:** Resilient broadcast channel handling catching Tokio stream lag to emit `sync-required` events, preventing silent socket disconnects on slow connections.
* **Declarative AEAD Cookie Sessions:** Cryptographically sealed client sessions utilizing `axum-extra`'s `PrivateCookieJar` with 512-bit SHA-512 master key expansion and zero runtime database lookup overhead.
* **Finite Board Capacity & Auto-Pruning:** Strictly enforced board limits (100 threads per board) with automated asynchronous cascade deletion of orphaned media files ($O(1)$ storage ceiling).
* **Chronological Bumping & Bump Limit:** Real-time thread bumping with `bumped_at` timestamps, chronological reply ordering via PostgreSQL `jsonb_agg`, `sage` bypass, and an automated 300-reply bump limit.
* **Classic & Secure Tripcode Engine:** Native parser for traditional (`#password`) and salted secure (`##password`) tripcodes with dedicated styling.
* **Pluggable Storage Abstraction:** Abstracted `StorageBackend` trait supporting local disk storage and cloud object stores (S3, MinIO, GCS) with non-blocking concurrent file writes and atomic thumbnail rollbacks.
* **High-Fidelity Fast Thumbnails:** High-performance integer downsampling using the `image` crate with strict dimension bomb defenses (10000x10000px validation), magic-byte format verification, and EXIF metadata stripping.
* **Zero-I/O Template Engine:** In-memory pre-cached MiniJinja template rendering compiled directly into RAM on startup for sub-millisecond page and item assembly.
* **Transactional Versioned Migrations:** Robust, run-once schema migrations (`schema_migrations` tracking table) executing within atomic transactions, eliminating boot-time `UPDATE` backfill bottlenecks.
* **IPv6 Subnet-Aware Rate Limiting:** Token-bucket IP rate limiter (2 req/sec, burst 10) with `/64` subnet masking to prevent rate-limit evasion through IPv6 address rotation.
* **Double-Submit CSRF Protection:** Timing-attack resistant CSRF middleware utilizing `subtle::ConstantTimeEq`, non-HttpOnly cookie distribution, HTML meta injection, and client-side `X-CSRF-Token` validation.
* **Pure Crockfordian JavaScript:** Modular client-side SPA runtime written with zero usage of `this`, `class`, `var`, `new` (in application code), or `void` operators.
* **Componentized Frontend Architecture:** Decomposed into dedicated ES modules (`post-renderer`, `tag-hover`, `reply-box`, `post-actions`, `post-form`) with explicit lifecycle teardowns to prevent memory leaks.
* **In-Place Image Expansion:** Clickable thumbnail expansion within the feed and thread views, with filename links directly opening raw full-resolution uploads in a new tab.
* **HTML5 History API Routing:** Clean URLs (`/g`, `/g/thread/a1b2c3d4e`, static views) with deep-linking support and History API client navigation.
* **Graceful Shutdown & Draining:** Integrated `SIGINT`/`SIGTERM` signal listening with active TCP connection draining for both cleartext and ALPN TLS HTTP/2 servers.
* **OWASP Hardened:** Includes strict Content Security Policy (CSP), Slowloris protection, XSS sanitization, timing-attack resistant password verification (`bcrypt`), and defensive security headers (`nosniff`, `DENY`, `mode=block`).

---

> **⚠️ Note on User Accounts / Authentication:**  
> The login and registration system (`/auth`, `src/routes/auth.rs`) is included strictly as a **functional demonstration** of the session management, private cookie encryption, and bcrypt capabilities. True to traditional imageboard culture, all board browsing, thread creation, and replying remain completely open, anonymous, and account-free by default.

---

## 🛠️ Tech Stack

* **Backend:** Rust (2024 Edition)
* **Web Framework:** [Axum 0.8](https://github.com/tokio-rs/axum) / [Axum-Extra 0.12](https://docs.rs/axum-extra) / [Tower](https://github.com/tower-rs/tower) / [Hyper 1.0](https://hyper.rs/)
* **Async Runtime:** [Tokio](https://tokio.rs/)
* **Database Driver:** [SQLx (PostgreSQL 16+)](https://github.com/launchbadge/sqlx)
* **Template Engine:** [MiniJinja](https://github.com/mitsuhiko/minijinja) (In-Memory Pre-cached)
* **Image Processing:** `image` crate (Fast integer downsampling & magic-byte sniffing)
* **Frontend:** Vanilla JavaScript (ES6 Modules, Crockfordian), HTML5, CSS3
* **Protocol:** HTTP/2 over TLS (ALPN `h2`) / Cleartext HTTP / Server-Sent Events (SSE)
* **Sessions:** AES-256-GCM Encrypted `PrivateCookieJar` (`axum-extra`)

---

## 📋 Prerequisites

* **Rust** `1.85+` (Cargo)
* **PostgreSQL** `16+` (or Docker)
* **OpenSSL** (optional, for local HTTP/2 TLS certificates)

---

## 🏁 Quick Start (Local Development)

### 1. Clone the Repository
```bash
git clone https://github.com/joncody/moarchan.git
cd moarchan
```

### 2. Create the Local PostgreSQL Database
Log into your local PostgreSQL CLI and create the database:
```sql
CREATE DATABASE moarchan;
```

### 3. Configure Environment Variables
Create a `.env` file in the root project directory:
```ini
PORT=9001
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=moarchan
POSTGRES_SSLMODE=disable
SESSION_HASH_KEY=12345678901234567890123456789012
SESSION_BLOCK_KEY=abcdefghijklmnopqrstuvwx12345678
UPLOAD_PATH=./static/images/uploads
UPLOAD_URL_PREFIX=/static/images/uploads
VIEWS_PATH=./static/views
```

*(Optional: For local ALPN HTTP/2 over TLS, generate self-signed certificates:)*
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=localhost"
```

### 4. Build and Run
```bash
cargo run --release
```

Navigate to `http://localhost:9001` (or `https://localhost:9001` if TLS certs are present) in your browser.

---

## 🐳 Running with Docker Compose

If you prefer running the application and database together in containerized environments:

```bash
docker-compose up --build
```

---

## ⚙️ Environment Configuration Reference

| Environment Variable | Default Value | Description |
| :--- | :--- | :--- |
| `PORT` | `9001` | Server HTTP port |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host address |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `postgres` | PostgreSQL username |
| `POSTGRES_PASSWORD` | `postgres` | PostgreSQL password |
| `POSTGRES_DB` | `moarchan` | Database name |
| `POSTGRES_SSLMODE` | `disable` | SSL mode (`disable`, `require`, `verify-full`) |
| `SESSION_HASH_KEY` | *(32 bytes)* | Secret key for deriving session master key |
| `SESSION_BLOCK_KEY` | *(32 bytes)* | Secret key for deriving session master key |
| `UPLOAD_PATH` | `./static/images/uploads` | Local filesystem base path for media storage |
| `UPLOAD_URL_PREFIX` | `/static/images/uploads` | Public URL prefix for uploaded media assets |
| `VIEWS_PATH` | `./static/views` | Directory path containing HTML templates |
| `TLS_CERT_PATH` | *(optional)* | Path to PEM-encoded TLS certificate file |
| `TLS_KEY_PATH` | *(optional)* | Path to PEM-encoded TLS private key file |

---

## 🏗️ Project Architecture

```
.
├── Cargo.toml            # Project dependencies & build manifest
├── docker-compose.yml    # Container orchestration setup
├── .env                  # Local environment configuration (git-ignored)
├── src/
│   ├── main.rs           # Application bootstrap, graceful shutdown & server launch
│   ├── config.rs         # Environment variable configuration loader
│   ├── state.rs          # Thread-safe global AppState & cookie key container
│   ├── error.rs          # Unified error handling & HTTP response conversion
│   ├── db/
│   │   ├── mod.rs        # DB module entrypoint
│   │   ├── migrations.rs # Versioned transactional schema migrations
│   │   └── queries.rs    # Domain SQL queries & aggregate builders
│   ├── middleware/
│   │   ├── mod.rs        # Middleware module entrypoint
│   │   ├── csrf.rs       # Double-submit cookie CSRF middleware
│   │   ├── rate_limit.rs # IPv6 /64 subnet-aware token-bucket rate limiter
│   │   └── security.rs   # Content Security Policy (CSP) & defensive headers
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
│   │       ├── replies.rs# Reply creation & bump limit endpoint
│   │       ├── delete.rs # Post/file deletion endpoint
│   │       └── stream.rs # Real-time SSE stream & lag recovery endpoint
│   ├── services/
│   │   ├── mod.rs        # Services module entrypoint
│   │   ├── auth.rs       # Bcrypt password verification & hashing
│   │   ├── image.rs      # Thumbnailing, magic-byte checking & EXIF stripping
│   │   ├── sanitizer.rs  # HTML escaping, tripcode engine & quote parsing
│   │   └── sse.rs        # Distributed Postgres LISTEN/NOTIFY SSE hub
│   └── storage/
│       ├── mod.rs        # Pluggable StorageBackend trait
│       └── local.rs      # Concurrent local filesystem storage implementation
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
    │       └── service.js# Imageboard thread/reply orchestrator
    └── views/            # MiniJinja HTML templates
```

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.
