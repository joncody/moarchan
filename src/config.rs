use std::env;

#[derive(Clone, Debug)]
pub struct AppConfig {
    pub port: u16,
    pub ssl_port: u16,
    pub tls_cert_path: Option<String>,
    pub tls_key_path: Option<String>,
    pub database_url: String,
    pub session_hash_key: String,
    pub session_block_key: String,
    pub upload_path: String,
    pub upload_url_prefix: String,
    pub views_path: String,
}

impl AppConfig {
    pub fn from_env() -> Self {
        let host = env::var("POSTGRES_HOST").unwrap_or_else(|_| "localhost".into());
        let port = env::var("POSTGRES_PORT").unwrap_or_else(|_| "5432".into());
        let user = env::var("POSTGRES_USER").unwrap_or_else(|_| "postgres".into());
        let pass = env::var("POSTGRES_PASSWORD").unwrap_or_else(|_| "postgres".into());
        let db = env::var("POSTGRES_DB").unwrap_or_else(|_| "moarchan".into());
        let sslmode = env::var("POSTGRES_SSLMODE").unwrap_or_else(|_| "disable".into());

        let database_url = format!(
            "postgres://{user}:{pass}@{host}:{port}/{db}?sslmode={sslmode}"
        );

        let views_path = env::var("VIEWS_PATH").unwrap_or_else(|_| {
            if std::path::Path::new("./static/views").exists() {
                "./static/views".into()
            } else if std::path::Path::new("./views").exists() {
                "./views".into()
            } else {
                "./static/views".into()
            }
        });

        Self {
            port: env::var("PORT").ok().and_then(|p| p.parse().ok()).unwrap_or(9001),
            ssl_port: env::var("SSL_PORT").ok().and_then(|p| p.parse().ok()).unwrap_or(0),
            tls_cert_path: env::var("TLS_CERT_PATH").ok(),
            tls_key_path: env::var("TLS_KEY_PATH").ok(),
            database_url,
            session_hash_key: env::var("SESSION_HASH_KEY")
                .unwrap_or_else(|_| "12345678901234567890123456789012".into()),
            session_block_key: env::var("SESSION_BLOCK_KEY")
                .unwrap_or_else(|_| "abcdefghijklmnopqrstuvwx12345678".into()),
            upload_path: env::var("UPLOAD_PATH").unwrap_or_else(|_| "./static/images/uploads".into()),
            upload_url_prefix: env::var("UPLOAD_URL_PREFIX").unwrap_or_else(|_| "/static/images/uploads".into()),
            views_path,
        }
    }
}
