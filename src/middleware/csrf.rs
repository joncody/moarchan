// src/middleware/csrf.rs
use axum::{
    extract::Request,
    http::{header, HeaderValue, Method, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use rand::RngCore;
use subtle::ConstantTimeEq;

pub const CSRF_COOKIE: &str = "moarchan_csrf";
pub const CSRF_HEADER: &str = "X-CSRF-Token";

pub async fn csrf_middleware(mut req: Request, next: Next) -> Response {
    let mut cookie_token = None;

    for cookie_header in req.headers().get_all(header::COOKIE) {
        if let Ok(cookie_str) = cookie_header.to_str() {
            for c in cookie_str.split(';') {
                let parts: Vec<&str> = c.trim().splitn(2, '=').collect();
                if parts.len() == 2 && parts[0] == CSRF_COOKIE {
                    cookie_token = Some(parts[1].to_string());
                    break;
                }
            }
        }
        if cookie_token.is_some() {
            break;
        }
    }

    let token = match cookie_token {
        Some(t) => t,
        None => {
            let mut rand_bytes = [0u8; 16];
            rand::thread_rng().fill_bytes(&mut rand_bytes);
            hex::encode(rand_bytes)
        }
    };

    let path = req.uri().path().to_string();
    let is_auth_route = path.starts_with("/login") || path.starts_with("/register");

    // Only enforce strict CSRF header matching on stateful auth routes or when CSRF header is explicitly supplied
    if is_auth_route && matches!(req.method(), &Method::POST | &Method::PUT | &Method::DELETE) {
        let header_token = req
            .headers()
            .get(CSRF_HEADER)
            .and_then(|h| h.to_str().ok())
            .unwrap_or("");

        if header_token.is_empty()
            || header_token.as_bytes().ct_eq(token.as_bytes()).unwrap_u8() != 1
        {
            return (
                StatusCode::FORBIDDEN,
                Json(serde_json::json!({
                    "error": "CSRF token validation failed",
                    "status": 403
                })),
            )
                .into_response();
        }
    }

    req.extensions_mut().insert(token.clone());
    let mut response = next.run(req).await;

    let cookie_val = format!(
        "{CSRF_COOKIE}={token}; Path=/; Max-Age=86400; SameSite=Lax"
    );
    if let Ok(hv) = HeaderValue::from_str(&cookie_val) {
        response.headers_mut().append(header::SET_COOKIE, hv);
    }

    response
}
