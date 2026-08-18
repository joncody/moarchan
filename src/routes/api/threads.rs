use axum::{
    extract::{Multipart, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use crate::{
    error::AppError,
    services::{
        auth::hash_password,
        image::process_upload,
        sanitizer::{generate_unique_identifiers, sanitize_comment},
    },
    state::AppState,
};

pub async fn create_thread_handler(
    State(state): State<AppState>,
    mut multipart: Multipart,
) -> Result<impl IntoResponse, AppError> {
    let mut topic = String::new();
    let mut name = String::new();
    let mut subject = String::new();
    let mut options = String::new();
    let mut raw_comment = String::new();
    let mut password = String::new();
    let mut file_data: Option<(String, Vec<u8>)> = None;

    while let Some(field) = multipart.next_field().await.map_err(|e| AppError::Multipart(e.to_string()))? {
        let field_name = field.name().unwrap_or("").to_string();
        match field_name.as_str() {
            "topic" => topic = field.text().await.unwrap_or_default().trim().to_string(),
            "name" => name = field.text().await.unwrap_or_default().trim().to_string(),
            "subject" => subject = field.text().await.unwrap_or_default().trim().to_string(),
            "options" => options = field.text().await.unwrap_or_default().trim().to_string(),
            "comment" => raw_comment = field.text().await.unwrap_or_default().trim().to_string(),
            "password" => password = field.text().await.unwrap_or_default().trim().to_string(),
            "file" => {
                let filename = field.file_name().unwrap_or("upload.jpg").to_string();
                let bytes = field.bytes().await.map_err(|e| AppError::Multipart(e.to_string()))?.to_vec();
                if !bytes.is_empty() {
                    file_data = Some((filename, bytes));
                }
            }
            _ => {}
        }
    }

    let clean_topic = topic.trim_matches('/').to_string();
    if clean_topic.is_empty() {
        return Err(AppError::BadRequest("Missing topic".into()));
    }
    let (orig_filename, raw_bytes) = file_data.ok_or_else(|| AppError::BadRequest("Image required for new threads".into()))?;

    let (timestamp, hash) = generate_unique_identifiers();
    let processed_image = process_upload(raw_bytes, &orig_filename, &hash, state.storage.as_ref()).await?;

    let (comment, _) = sanitize_comment(&raw_comment);
    let final_name = if name.is_empty() { "Anonymous".to_string() } else { html_escape::encode_safe(&name).to_string() };
    let final_subject = html_escape::encode_safe(&subject).to_string();
    let pass_hash = hash_password(&password).await?;

    sqlx::query(
        r#"
        INSERT INTO threads (hash, topic, name, subject, options, password_hash, comment, file_name, file_mime, file_size, file_dimensions, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        "#
    )
    .bind(&hash)
    .bind(&clean_topic)
    .bind(&final_name)
    .bind(&final_subject)
    .bind(&options)
    .bind(&pass_hash)
    .bind(&comment)
    .bind(&processed_image.details.name)
    .bind(&processed_image.details.mime)
    .bind(&processed_image.details.size)
    .bind(&processed_image.details.dimensions)
    .bind(&timestamp)
    .execute(&state.db)
    .await?;

    let mut thread_json = serde_json::json!({
        "hash": hash,
        "topic": clean_topic,
        "name": final_name,
        "subject": final_subject,
        "options": options,
        "comment": comment,
        "file_name": processed_image.details.name,
        "file_mime": processed_image.details.mime,
        "file_size": processed_image.details.size,
        "file_dimensions": processed_image.details.dimensions,
        "timestamp": timestamp,
        "replies": [],
        "taggedBy": [],
        "tagging": []
    });

    match state.render_template("thread-item", &thread_json) {
        Ok(rendered) => {
            thread_json["html"] = serde_json::Value::String(rendered);
        }
        Err(e) => {
            tracing::error!("Failed to pre-render thread-item for SSE: {e:#?}");
        }
    }

    state.sse_hub.broadcast(&state.db, &clean_topic, "new-thread", thread_json.clone()).await;

    Ok((StatusCode::CREATED, Json(thread_json)))
}
