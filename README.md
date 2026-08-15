# moarchan

A high-performance, real-time 4chan clone built from scratch using **Go**, **HTTP/2**, **Server-Sent Events (SSE)**, **PostgreSQL (`LISTEN / NOTIFY`)**, and **Vanilla JavaScript** (strictly following Douglas Crockford's coding standards).

---

## 🚀 Key Features

* **HTTP/2 & Real-Time SSE Bus:** Multiplexed Server-Sent Events distributed horizontally across Go instances using PostgreSQL `LISTEN / NOTIFY` with compact JSON payloads.
* **Pure Crockfordian JavaScript:** Modular client-side SPA runtime written with zero usage of `this`, `class`, `var`, `new` (in application code), or `void` operators.
* **Componentized Frontend Architecture:** Decomposed into dedicated ES modules (`post-renderer`, `tag-hover`, `reply-box`, `post-actions`, `post-form`) with explicit lifecycle teardowns to prevent memory leaks.
* **In-Place Image Expansion:** Clickable thumbnail expansion within the feed and thread views, with filename links directly opening raw full-resolution uploads in a new tab.
* **Chronological Bumping & Sage:** Real-time thread bumping with `bumped_at` timestamps, chronological reply ordering via `jsonb_agg`, and `sage` bypass support.
* **HTML5 History API Routing:** Clean URLs (`/g`, `/g/thread/a1b2c3d4e`) with deep-linking support and History API client navigation.
* **Decoupled Engine (`frame`):** Core framework provides session management, middleware pipelines, dynamic routing, and an SSE event hub independent of domain logic.
* **Protected Deletions & File Cleanup:** Post and file deletion secured with bcrypt password verification (or admin override), with automatic disk cleanup of orphaned cascade files.
* **OWASP Hardened:** Includes Slowloris protection (`ReadHeaderTimeout`), image dimension bombs defense (10000x10000px bounds validation), XSS sanitization, timing-attack resistant password verification (`bcrypt`), and security headers.

---

> **⚠️ Note on User Accounts / Authentication:**  
> The login and registration system (`/auth`, `frame/auth.go`) is included strictly as a **functional demonstration** of the underlying `frame` framework's session management, cookie encryption, and bcrypt capabilities. True to traditional imageboard culture, all board browsing, thread creation, and replying remain completely open, anonymous, and account-free by default.

---

## 🛠️ Tech Stack

* **Backend:** Go (Golang)
* **Frontend:** Vanilla JavaScript (ES6 Modules), HTML5, CSS3
* **Database:** PostgreSQL 16+
* **Protocol:** HTTP/2 over TLS / Cleartext HTTP/2 (h2c) / Server-Sent Events
* **Routing:** `github.com/gorilla/mux`
* **Sessions:** `github.com/gorilla/sessions`

---

## 📋 Prerequisites

* **Go** `1.22+`
* **PostgreSQL** `16+` (or Docker)

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
```

### 4. Run the Application
```bash
go run .
```

Navigate to `http://localhost:9001` in your browser.

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
| `PORT` | `9001` | Server port |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host address |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `postgres` | PostgreSQL username |
| `POSTGRES_PASSWORD` | `postgres` | PostgreSQL password |
| `POSTGRES_DB` | `moarchan` | Database name |
| `POSTGRES_SSLMODE` | `disable` | SSL mode (`disable`, `require`, `verify-full`) |
| `SESSION_HASH_KEY` | *(32 bytes)* | Secret key for signing session cookies |
| `SESSION_BLOCK_KEY` | *(32 bytes)* | Secret key for encrypting session payload |

---

## 🏗️ Project Architecture

```
.
├── main.go               # Application entrypoint & HTTP REST handlers
├── db_store.go           # Relational DB schema, migrations & queries
├── docker-compose.yml    # Container orchestration setup
├── .env                  # Local environment configuration (git-ignored)
├── frame/                # Standalone micro-framework
│   ├── frame.go          # Core app lifecycle & DB connection pool
│   ├── routes.go         # Fluent route builder & template renderer
│   ├── sse.go            # Distributed Postgres LISTEN/NOTIFY SSE hub
│   ├── middleware.go     # Logging, Panic Recovery, Security headers
│   ├── db.go             # Generic KV document database helpers
│   ├── auth.go           # Bcrypt user authentication (Demo)
│   └── session.go        # Secure Gorilla cookie sessions
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
    └── views/            # Go HTML templates
```

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.
