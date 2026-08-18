use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),

    #[error("Image processing error: {0}")]
    Image(String),

    #[error("Multipart error: {0}")]
    Multipart(String),

    #[error("Validation error: {0}")]
    BadRequest(String),

    #[error("Authentication required")]
    Unauthorized(String),

    #[error("Access denied: {0}")]
    Forbidden(String),

    #[error("Resource not found: {0}")]
    NotFound(String),

    #[error("Payload exceeded limit: {0}")]
    PayloadTooLarge(String),

    #[allow(dead_code)]
    #[error("Rate limit exceeded. Please slow down.")]
    RateLimited,

    #[error("Template error: {0}")]
    Template(#[from] minijinja::Error),

    #[error("Internal server error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match &self {
            AppError::BadRequest(msg) => (StatusCode::BAD_REQUEST, msg.clone()),
            AppError::Multipart(msg) => (StatusCode::BAD_REQUEST, msg.clone()),
            AppError::Unauthorized(msg) => (StatusCode::UNAUTHORIZED, msg.clone()),
            AppError::Forbidden(msg) => (StatusCode::FORBIDDEN, msg.clone()),
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg.clone()),
            AppError::RateLimited => (StatusCode::TOO_MANY_REQUESTS, self.to_string()),
            AppError::PayloadTooLarge(msg) => (StatusCode::PAYLOAD_TOO_LARGE, msg.clone()),
            AppError::Image(msg) => (StatusCode::BAD_REQUEST, msg.clone()),
            AppError::Database(err) => {
                tracing::error!("Database query failed: {:?}", err);
                (StatusCode::INTERNAL_SERVER_ERROR, "Database error".into())
            }
            AppError::Template(err) => {
                tracing::error!("Template error: {:#}", err);
                (StatusCode::INTERNAL_SERVER_ERROR, format!("Rendering error: {err}"))
            }
            AppError::Internal(err) => {
                tracing::error!("Internal system error: {:?}", err);
                (StatusCode::INTERNAL_SERVER_ERROR, "Internal server error".into())
            }
        };

        let body = Json(json!({
            "error": message,
            "status": status.as_u16()
        }));

        (status, body).into_response()
    }
}
