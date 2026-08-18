use axum::http::HeaderMap;
use minijinja::Environment;
use sqlx::PgPool;
use std::sync::Arc;
use crate::{
    config::AppConfig,
    middleware::rate_limit::IpRateLimiter,
    models::auth::SessionUser,
    services::{session::SessionStore, sse::SseHub},
    storage::StorageBackend,
};

#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    pub storage: Arc<dyn StorageBackend>,
    pub sse_hub: Arc<SseHub>,
    pub rate_limiter: Arc<IpRateLimiter>,
    pub templates: Arc<Environment<'static>>,
    pub session_store: Arc<SessionStore>,
    #[allow(dead_code)]
    pub config: Arc<AppConfig>,
}

impl AppState {
    pub fn get_template(&self, name: &str) -> Result<minijinja::Template<'_, '_>, minijinja::Error> {
        if let Ok(tmpl) = self.templates.get_template(name) {
            return Ok(tmpl);
        }
        if !name.ends_with(".html") {
            let html_name = format!("{name}.html");
            if let Ok(tmpl) = self.templates.get_template(&html_name) {
                return Ok(tmpl);
            }
        }
        if !name.ends_with(".tmpl") {
            let tmpl_name = format!("{name}.tmpl");
            if let Ok(tmpl) = self.templates.get_template(&tmpl_name) {
                return Ok(tmpl);
            }
        }
        self.templates.get_template(name)
    }

    pub fn render_template(&self, name: &str, ctx: &serde_json::Value) -> Result<String, minijinja::Error> {
        let tmpl = self.get_template(name)?;
        tmpl.render(ctx)
    }

    pub fn extract_session(&self, headers: &HeaderMap) -> Option<SessionUser> {
        let cookie_header = headers.get(axum::http::header::COOKIE)?.to_str().ok()?;
        for cookie in cookie_header.split(';') {
            let parts: Vec<&str> = cookie.trim().splitn(2, '=').collect();
            if parts.len() == 2 && parts[0] == self.session_store.cookie_name {
                if let Ok(map) = self.session_store.decrypt(parts[1]) {
                    if let (Some(alias), Some(privilege)) = (map.get("alias"), map.get("privilege")) {
                        return Some(SessionUser {
                            alias: alias.clone(),
                            privilege: privilege.clone(),
                        });
                    }
                }
            }
        }
        None
    }
}
