pub mod api;
pub mod auth;
pub mod pages;

use axum::{
    http::StatusCode,
    middleware::{from_fn, from_fn_with_state},
    routing::{get, post},
    Router,
};
use tower_http::{cors::CorsLayer, services::ServeDir, trace::TraceLayer};
use crate::{
    middleware::{
        csrf::csrf_middleware,
        rate_limit::rate_limit_middleware,
        security::security_headers_middleware,
    },
    routes::{
        api::{
            delete::delete_post_handler,
            replies::create_reply_handler,
            stream::sse_stream_handler,
            threads::create_thread_handler,
        },
        auth::{login_handler, logout_handler, register_handler},
        pages::{base_shell_handler, render_spa_handler},
    },
    state::AppState,
};

pub fn build_router(state: AppState) -> Router {
    let api_routes = Router::new()
        .route("/threads", post(create_thread_handler))
        .route("/replies", post(create_reply_handler))
        .route("/posts/delete", post(delete_post_handler))
        .route("/stream", get(sse_stream_handler))
        .route("/render", get(render_spa_handler));

    Router::new()
        .nest("/api", api_routes)
        .route("/login", post(login_handler))
        .route("/register", post(register_handler))
        .route("/logout", post(logout_handler))
        .route("/favicon.ico", get(|| async { StatusCode::NO_CONTENT }))
        .nest_service("/static", ServeDir::new("./static"))
        .fallback(base_shell_handler)
        .layer(from_fn(csrf_middleware))
        .layer(from_fn_with_state(state.rate_limiter.clone(), rate_limit_middleware))
        .layer(from_fn(security_headers_middleware))
        .layer(TraceLayer::new_for_http())
        .layer(CorsLayer::permissive())
        .with_state(state)
}
