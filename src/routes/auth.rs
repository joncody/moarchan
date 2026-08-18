use axum::{
    extract::{FromRequest, Multipart, Request, State},
    http::{header, HeaderMap, HeaderValue, StatusCode},
    response::IntoResponse,
    Json,
};
use serde::Deserialize;
use sqlx::Row;
use std::collections::HashMap;
use crate::{
    error::AppError,
    models::auth::AuthRecord,
    services::auth::{hash_account_password, verify_password},
    state::AppState,
};

#[derive(Default, Deserialize, Debug)]
pub struct AuthForm {
    pub alias: String,
    pub password: String,
}

async fn extract_auth_form(headers: &HeaderMap, req: Request, state: &AppState) -> Result<AuthForm, AppError> {
    let content_type = headers
        .get(header::CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_lowercase();

    let mut form = AuthForm::default();

    if content_type.starts_with("multipart/form-data") {
        let mut multipart = Multipart::from_request(req, state)
            .await
            .map_err(|e| AppError::Multipart(e.to_string()))?;

        while let Some(field) = multipart.next_field().await.map_err(|e| AppError::Multipart(e.to_string()))? {
            let name = field.name().unwrap_or("").to_string();
            let text = field.text().await.unwrap_or_default().trim().to_string();

            match name.as_str() {
                "alias" | "username" | "user" => form.alias = text,
                "password" | "pass" | "pwd" => form.password = text,
                _ => {}
            }
        }
    } else {
        let bytes = axum::body::to_bytes(req.into_body(), 1024 * 64)
            .await
            .map_err(|e| AppError::BadRequest(e.to_string()))?;

        if content_type.starts_with("application/json") {
            if let Ok(f) = serde_json::from_slice::<AuthForm>(&bytes) {
                form = f;
            }
        } else {
            let body_str = String::from_utf8_lossy(&bytes);
            for pair in body_str.split('&') {
                let mut parts = pair.splitn(2, '=');
                if let (Some(k), Some(v)) = (parts.next(), parts.next()) {
                    let k = percent_decode(k);
                    let v = percent_decode(v);
                    match k.as_str() {
                        "alias" | "username" | "user" => form.alias = v,
                        "password" | "pass" | "pwd" => form.password = v,
                        _ => {}
                    }
                }
            }
        }
    }

    Ok(form)
}

fn percent_decode(input: &str) -> String {
    let replaced = input.replace('+', " ");
    let mut bytes = Vec::new();
    let mut chars = replaced.chars().peekable();

    while let Some(ch) = chars.next() {
        if ch == '%' {
            let h1 = chars.next();
            let h2 = chars.next();
            if let (Some(c1), Some(c2)) = (h1, h2) {
                if let Ok(b) = u8::from_str_radix(&format!("{c1}{c2}"), 16) {
                    bytes.push(b);
                    continue;
                }
            }
        }
        bytes.extend_from_slice(ch.encode_utf8(&mut [0; 4]).as_bytes());
    }

    String::from_utf8_lossy(&bytes).into_owned()
}

pub async fn register_handler(
    State(state): State<AppState>,
    headers: HeaderMap,
    req: Request,
) -> Result<impl IntoResponse, AppError> {
    let form = extract_auth_form(&headers, req, &state).await?;

    if form.alias.len() < 3 || form.password.len() < 8 {
        return Err(AppError::BadRequest("Alias must be 3+ chars, password 8+ chars".into()));
    }

    let hashed = hash_account_password(&form.password).await?;
    let record = AuthRecord {
        password_hash: hashed,
        privilege: "user".into(),
    };
    let json_data = serde_json::to_value(&record).unwrap();

    sqlx::query(
        "INSERT INTO auth (key, value) VALUES ($1, $2)"
    )
    .bind(&form.alias)
    .bind(&json_data)
    .execute(&state.db)
    .await
    .map_err(|_| AppError::BadRequest("Alias already taken".into()))?;

    let mut session_map = HashMap::new();
    session_map.insert("alias".into(), form.alias);
    session_map.insert("privilege".into(), "user".into());
    let token = state.session_store.encrypt(&session_map)?;

    let cookie = state.session_store.cookie_value(&token);
    let mut resp_headers = HeaderMap::new();
    resp_headers.insert(header::SET_COOKIE, HeaderValue::from_str(&cookie).unwrap());

    Ok((StatusCode::OK, resp_headers, Json(serde_json::json!({"status": "ok"}))))
}

pub async fn login_handler(
    State(state): State<AppState>,
    headers: HeaderMap,
    req: Request,
) -> Result<impl IntoResponse, AppError> {
    let form = extract_auth_form(&headers, req, &state).await?;

    let row = sqlx::query("SELECT value FROM auth WHERE key = $1")
        .bind(&form.alias)
        .fetch_optional(&state.db)
        .await?;

    let mut valid = false;
    let mut privilege = "user".to_string();

    if let Some(r) = row {
        let val: serde_json::Value = r.try_get("value")?;
        if let Ok(rec) = serde_json::from_value::<AuthRecord>(val) {
            privilege = rec.privilege;
            valid = verify_password(Some(&rec.password_hash), &form.password).await?;
        }
    } else {
        let _ = verify_password(None, &form.password).await?;
    }

    if !valid {
        return Err(AppError::Unauthorized("Invalid username or password".into()));
    }

    let mut session_map = HashMap::new();
    session_map.insert("alias".into(), form.alias);
    session_map.insert("privilege".into(), privilege);
    let token = state.session_store.encrypt(&session_map)?;

    let cookie = state.session_store.cookie_value(&token);
    let mut resp_headers = HeaderMap::new();
    resp_headers.insert(header::SET_COOKIE, HeaderValue::from_str(&cookie).unwrap());

    Ok((StatusCode::OK, resp_headers, Json(serde_json::json!({"status": "ok"}))))
}

pub async fn logout_handler(State(state): State<AppState>) -> impl IntoResponse {
    let cookie = state.session_store.delete_cookie_value();
    let mut headers = HeaderMap::new();
    headers.insert(header::SET_COOKIE, HeaderValue::from_str(&cookie).unwrap());
    headers.insert(header::LOCATION, HeaderValue::from_static("/"));
    (StatusCode::SEE_OTHER, headers)
}
