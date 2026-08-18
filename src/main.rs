mod config;
mod db;
mod error;
mod middleware;
mod models;
mod routes;
mod services;
mod state;
mod storage;

use axum_server::tls_rustls::RustlsConfig;
use minijinja::Environment;
use sqlx::postgres::PgPoolOptions;
use std::{net::SocketAddr, path::PathBuf, sync::Arc, time::Duration};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::{
    config::AppConfig,
    db::migrations::run_migrations,
    middleware::rate_limit::IpRateLimiter,
    routes::build_router,
    services::{session::SessionStore, sse::SseHub},
    state::AppState,
    storage::local::LocalDiskStorage,
};

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to install Ctrl+C signal handler");
    };

    #[cfg(unix)]
    let terminate = async {
        if let Ok(mut stream) = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate()) {
            stream.recv().await;
        }
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }

    tracing::info!("Shutdown signal received, draining active connections...");
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. Structured Logging Setup (Shows latency timers while suppressing socket trace noise)
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "moarchan=debug,tower_http=debug,axum=info".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    // 2. Load Configuration
    let config = AppConfig::from_env();
    tracing::info!("Bootstrapping MoarChan engine on port {}", config.port);

    // 3. Lock-Free Database Connection Pool
    let pool = PgPoolOptions::new()
        .max_connections(25)
        .min_connections(5)
        .acquire_timeout(Duration::from_secs(5))
        .idle_timeout(Duration::from_secs(300))
        .connect(&config.database_url)
        .await
        .map_err(|e| format!("Failed to connect to PostgreSQL ({}): {e}", config.database_url))?;

    // 4. Run-Once Versioned Schema Migrations
    run_migrations(&pool).await?;

    // 5. Pluggable Storage Backend
    let storage = LocalDiskStorage::new(&config.upload_path, &config.upload_url_prefix).await?;

    // 6. Template Engine Configuration: In-Memory RAM Pre-caching (Zero Disk I/O)
    let mut env = Environment::new();
    env.set_undefined_behavior(minijinja::UndefinedBehavior::Chainable);

    if let Ok(views_dir) = std::fs::read_dir(&config.views_path) {
        for entry in views_dir.flatten() {
            let path = entry.path();
            if path.is_file() {
                if let Ok(content) = std::fs::read_to_string(&path) {
                    let file_name = path.file_name().and_then(|s| s.to_str()).unwrap_or("").to_string();
                    let stem = path.file_stem().and_then(|s| s.to_str()).unwrap_or("").to_string();
                    if !file_name.is_empty() {
                        if let Err(e) = env.add_template_owned(file_name.clone(), content.clone()) {
                            tracing::error!("Failed to parse template '{}': {}", file_name, e);
                        }
                    }
                    if !stem.is_empty() {
                        let _ = env.add_template_owned(stem, content);
                    }
                }
            }
        }
    }

    env.add_function("tokey", |val: minijinja::Value| {
        val.as_str().unwrap_or("").to_lowercase().replace(' ', "-").replace("---", "-")
    });
    env.add_filter("tokey", |val: minijinja::Value| {
        val.as_str().unwrap_or("").to_lowercase().replace(' ', "-").replace("---", "-")
    });
    env.add_filter("unescaped", |val: minijinja::Value| {
        minijinja::Value::from_safe_string(val.to_string())
    });
    env.add_filter("unescape", |val: minijinja::Value| {
        minijinja::Value::from_safe_string(val.to_string())
    });

    // 7. Initialize Application State Container
    let sse_hub = Arc::new(SseHub::new());
    let rate_limiter = Arc::new(IpRateLimiter::new(2.0, 10)); // 2 req/sec, burst 10
    let session_store = Arc::new(SessionStore::new(
        "moarchan",
        &config.session_hash_key,
        &config.session_block_key,
        config.ssl_port > 0,
    ));

    let state = AppState {
        db: pool.clone(),
        storage: Arc::new(storage),
        sse_hub: sse_hub.clone(),
        rate_limiter,
        templates: Arc::new(env),
        session_store,
        config: Arc::new(config.clone()),
    };

    // 8. Start Distributed Postgres LISTEN / NOTIFY SSE Hub Worker
    SseHub::start_postgres_listener(state.clone());

    // 9. Build Master Axum Router
    let app = build_router(state);

    // 10. HTTP/2 Server Launch with Graceful Shutdown
    let addr = SocketAddr::from(([0, 0, 0, 0], config.port));

    let cert_file = config.tls_cert_path.as_deref().or_else(|| {
        if std::path::Path::new("./cert.pem").exists() { Some("./cert.pem") } else { None }
    });
    let key_file = config.tls_key_path.as_deref().or_else(|| {
        if std::path::Path::new("./key.pem").exists() { Some("./key.pem") } else { None }
    });

    if let (Some(cert_path), Some(key_path)) = (cert_file, key_file) {
        tracing::info!("Configuring ALPN HTTP/2 over TLS (rustls)");
        let tls_config = RustlsConfig::from_pem_file(
            PathBuf::from(cert_path),
            PathBuf::from(key_path),
        )
        .await?;

        let handle = axum_server::Handle::new();
        let handle_clone = handle.clone();
        tokio::spawn(async move {
            shutdown_signal().await;
            handle_clone.graceful_shutdown(Some(Duration::from_secs(10)));
        });

        tracing::info!("MoarChan HTTP/2 server listening on https://{}", addr);
        axum_server::bind_rustls(addr, tls_config)
            .handle(handle)
            .serve(app.into_make_service_with_connect_info::<SocketAddr>())
            .await?;
    } else {
        tracing::info!("Starting Axum cleartext server on http://{}", addr);
        let listener = tokio::net::TcpListener::bind(addr).await?;
        axum::serve(listener, app.into_make_service_with_connect_info::<SocketAddr>())
            .with_graceful_shutdown(shutdown_signal())
            .await?;
    }

    tracing::info!("Server shut down gracefully.");
    Ok(())
}
