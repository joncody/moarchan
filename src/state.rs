use axum::extract::FromRef;
use axum_extra::extract::cookie::{Key, PrivateCookieJar};
use minijinja::Environment;
use sqlx::PgPool;
use std::sync::Arc;
use crate::{
    config::AppConfig,
    middleware::rate_limit::IpRateLimiter,
    models::auth::SessionUser,
    services::sse::SseHub,
    storage::StorageBackend,
};

pub const SESSION_COOKIE_NAME: &str = "moarchan";

#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    pub storage: Arc<dyn StorageBackend>,
    pub sse_hub: Arc<SseHub>,
    pub rate_limiter: Arc<IpRateLimiter>,
    pub templates: Arc<Environment<'static>>,
    pub cookie_key: Key,
    #[allow(dead_code)]
    pub config: Arc<AppConfig>,
}

impl FromRef<AppState> for Key {
    fn from_ref(state: &AppState) -> Self {
        state.cookie_key.clone()
    }
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

    pub fn extract_session(&self, jar: &PrivateCookieJar) -> Option<SessionUser> {
        let cookie = jar.get(SESSION_COOKIE_NAME)?;
        serde_json::from_str(cookie.value()).ok()
    }
}
