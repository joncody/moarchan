// src/middleware/security.rs
use axum::{
    extract::Request,
    http::header,
    middleware::Next,
    response::Response,
};

pub async fn security_headers_middleware(req: Request, next: Next) -> Response {
    let mut resp = next.run(req).await;
    let h = resp.headers_mut();
    h.insert(header::X_CONTENT_TYPE_OPTIONS, "nosniff".parse().unwrap());
    h.insert(header::X_FRAME_OPTIONS, "DENY".parse().unwrap());
    h.insert(header::X_XSS_PROTECTION, "1; mode=block".parse().unwrap());
    resp
}
