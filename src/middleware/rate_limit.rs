use axum::{
    extract::{ConnectInfo, Request, State},
    http::{Method, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use dashmap::DashMap;
use std::{
    net::SocketAddr,
    sync::Arc,
    time::{Duration, Instant},
};

#[derive(Clone, Copy)]
struct Bucket {
    tokens: f64,
    last_check: Instant,
}

pub struct IpRateLimiter {
    buckets: DashMap<String, Bucket>,
    rate: f64, // tokens added per second
    burst: f64,
}

impl IpRateLimiter {
    pub fn new(rate: f64, burst: usize) -> Self {
        let limiter = Self {
            buckets: DashMap::new(),
            rate,
            burst: burst as f64,
        };

        // Background cleanup worker
        let buckets_clone = limiter.buckets.clone();
        tokio::spawn(async move {
            loop {
                tokio::time::sleep(Duration::from_secs(300)).await;
                let now = Instant::now();
                buckets_clone.retain(|_, v| now.duration_since(v.last_check) < Duration::from_secs(300));
            }
        });

        limiter
    }

    pub fn allow(&self, ip: &str) -> bool {
        let mut entry = self.buckets.entry(ip.to_string()).or_insert_with(|| Bucket {
            tokens: self.burst - 1.0,
            last_check: Instant::now(),
        });

        let now = Instant::now();
        let elapsed = now.duration_since(entry.last_check).as_secs_f64();
        entry.tokens = (entry.tokens + elapsed * self.rate).min(self.burst);
        entry.last_check = now;

        if entry.tokens < 1.0 {
            false
        } else {
            entry.tokens -= 1.0;
            true
        }
    }
}

pub async fn rate_limit_middleware(
    State(limiter): State<Arc<IpRateLimiter>>,
    req: Request,
    next: Next,
) -> Response {
    if matches!(req.method(), &Method::POST | &Method::PUT | &Method::DELETE) {
        let socket_ip = req
            .extensions()
            .get::<ConnectInfo<SocketAddr>>()
            .map(|ci| ci.0.ip())
            .or_else(|| {
                req.extensions()
                    .get::<SocketAddr>()
                    .map(|addr| addr.ip())
            });

        let ip = if let Some(sock_ip) = socket_ip {
            if sock_ip.is_loopback() {
                req.headers()
                    .get("x-forwarded-for")
                    .and_then(|v| v.to_str().ok())
                    .map(|s| s.split(',').next().unwrap_or("").trim().to_string())
                    .or_else(|| {
                        req.headers()
                            .get("x-real-ip")
                            .and_then(|v| v.to_str().ok())
                            .map(String::from)
                    })
                    .unwrap_or_else(|| sock_ip.to_string())
            } else {
                sock_ip.to_string()
            }
        } else {
            req.headers()
                .get("x-forwarded-for")
                .and_then(|v| v.to_str().ok())
                .map(|s| s.split(',').next().unwrap_or("").trim().to_string())
                .or_else(|| {
                    req.headers()
                        .get("x-real-ip")
                        .and_then(|v| v.to_str().ok())
                        .map(String::from)
                })
                .unwrap_or_else(|| "127.0.0.1".to_string())
        };

        if !limiter.allow(&ip) {
            return (
                StatusCode::TOO_MANY_REQUESTS,
                [("Retry-After", "60")],
                Json(serde_json::json!({
                    "error": "Rate limit exceeded. Please slow down.",
                    "status": 429
                })),
            )
                .into_response();
        }
    }

    next.run(req).await
}
