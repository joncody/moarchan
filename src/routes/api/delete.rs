use axum::{
    extract::{FromRequest, Multipart, Request, State},
    http::{header, HeaderMap},
    response::IntoResponse,
    Json,
};
use crate::{
    db::queries::delete_post_or_file,
    error::AppError,
    state::AppState,
};

pub async fn delete_post_handler(
    State(state): State<AppState>,
    headers: HeaderMap,
    req: Request,
) -> Result<impl IntoResponse, AppError> {
    let is_admin = state.extract_session(&headers).map(|s| s.privilege == "admin").unwrap_or(false);

    let content_type = headers
        .get(header::CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_lowercase();

    let mut hashes = Vec::new();
    let mut password = String::new();
    let mut file_only = false;

    if content_type.starts_with("multipart/form-data") {
        let mut multipart = Multipart::from_request(req, &state)
            .await
            .map_err(|e| AppError::Multipart(e.to_string()))?;

        while let Some(field) = multipart.next_field().await.map_err(|e| AppError::Multipart(e.to_string()))? {
            let name = field.name().unwrap_or("").to_string();
            let text = field.text().await.unwrap_or_default().trim().to_string();

            match name.as_str() {
                "hash" | "onlypost" | "post" | "id" => {
                    if !text.is_empty() {
                        hashes.push(text);
                    }
                }
                "password" | "pwd" => {
                    password = text;
                }
                "file_only" | "fileonly" | "file-only" => {
                    if text == "true" || text == "on" || text == "1" {
                        file_only = true;
                    }
                }
                _ => {
                    if (text == "on" || text == "true" || text == "1") && !name.is_empty() && name != "delete" && name != "enter" {
                        hashes.push(name);
                    }
                }
            }
        }
    } else {
        let bytes = axum::body::to_bytes(req.into_body(), 1024 * 64)
            .await
            .map_err(|e| AppError::BadRequest(e.to_string()))?;

        if content_type.starts_with("application/json") {
            if let Ok(val) = serde_json::from_slice::<serde_json::Value>(&bytes) {
                if let Some(h) = val.get("hash").and_then(|v| v.as_str()) {
                    hashes.push(h.to_string());
                }
                if let Some(arr) = val.get("hashes").and_then(|v| v.as_array()) {
                    for item in arr {
                        if let Some(h) = item.as_str() {
                            hashes.push(h.to_string());
                        }
                    }
                }
                if let Some(p) = val.get("password").and_then(|v| v.as_str()) {
                    password = p.to_string();
                }
                if let Some(fo) = val.get("file_only").or_else(|| val.get("fileonly")).or_else(|| val.get("file-only")) {
                    file_only = fo.as_bool().unwrap_or(fo.as_str().map(|s| s == "true" || s == "on" || s == "1").unwrap_or(false));
                }
            }
        } else {
            let body_str = String::from_utf8_lossy(&bytes);
            for pair in body_str.split('&') {
                let mut parts = pair.splitn(2, '=');
                if let (Some(k), Some(v)) = (parts.next(), parts.next()) {
                    let k = percent_decode(k);
                    let v = percent_decode(v);
                    match k.as_str() {
                        "hash" | "onlypost" | "post" | "id" => {
                            if !v.is_empty() {
                                hashes.push(v);
                            }
                        }
                        "password" | "pwd" => password = v,
                        "file_only" | "fileonly" | "file-only" => {
                            if v == "true" || v == "on" || v == "1" {
                                file_only = true;
                            }
                        }
                        _ => {
                            if (v == "on" || v == "true" || v == "1") && !k.is_empty() && k != "delete" && k != "enter" {
                                hashes.push(k);
                            }
                        }
                    }
                }
            }
        }
    }

    if hashes.is_empty() {
        return Err(AppError::BadRequest("No post hash provided for deletion".into()));
    }

    let mut deleted_hashes = Vec::new();
    for hash in &hashes {
        match delete_post_or_file(&state.db, hash, &password, is_admin, file_only).await {
            Ok(info) => {
                for file_name in &info.file_names {
                    let _ = state.storage.delete(file_name).await;
                    let _ = state.storage.delete(&format!("thumb_{file_name}")).await;
                }

                let clean_topic = info.topic.trim_matches('/').to_string();
                let event_payload = serde_json::json!({
                    "hash": info.hash,
                    "topic": clean_topic,
                    "file_only": file_only
                });
                state.sse_hub.broadcast(&state.db, &clean_topic, "delete-post", event_payload).await;
                deleted_hashes.push(info.hash);
            }
            Err(e) => {
                if hashes.len() == 1 {
                    return Err(e);
                }
            }
        }
    }

    Ok(Json(serde_json::json!({
        "status": "ok",
        "deleted": deleted_hashes,
        "file_only": file_only
    })))
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
