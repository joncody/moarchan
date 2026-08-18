use axum::{
    extract::{Multipart, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use crate::{
    error::AppError,
    models::post::FileDetails,
    services::{
        auth::hash_password,
        image::process_upload,
        sanitizer::{generate_unique_identifiers, sanitize_comment},
    },
    state::AppState,
};

pub async fn create_reply_handler(
    State(state): State<AppState>,
    mut multipart: Multipart,
) -> Result<impl IntoResponse, AppError> {
    let mut topic = String::new();
    let mut thread_hash = String::new();
    let mut name = String::new();
    let mut options = String::new();
    let mut raw_comment = String::new();
    let mut password = String::new();
    let mut file_data: Option<(String, Vec<u8>)> = None;

    while let Some(field) = multipart.next_field().await.map_err(|e| AppError::Multipart(e.to_string()))? {
        let field_name = field.name().unwrap_or("").to_string();
        match field_name.as_str() {
            "topic" => topic = field.text().await.unwrap_or_default().trim().to_string(),
            "thread" => thread_hash = field.text().await.unwrap_or_default().trim().to_string(),
            "name" => name = field.text().await.unwrap_or_default().trim().to_string(),
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
    if clean_topic.is_empty() || thread_hash.is_empty() {
        return Err(AppError::BadRequest("Missing topic or thread hash".into()));
    }
    if raw_comment.is_empty() {
        return Err(AppError::BadRequest("Comment is required".into()));
    }

    let (timestamp, hash) = generate_unique_identifiers();

    let file_details = if let Some((orig_filename, raw_bytes)) = file_data {
        process_upload(raw_bytes, &orig_filename, &hash, state.storage.as_ref()).await?.details
    } else {
        FileDetails::default()
    };

    let (comment, tags) = sanitize_comment(&raw_comment);
    let final_name = if name.is_empty() { "Anonymous".to_string() } else { html_escape::encode_safe(&name).to_string() };
    let pass_hash = hash_password(&password).await?;

    let mut tx = state.db.begin().await?;

    let exists: bool = sqlx::query_scalar(
        "SELECT EXISTS(SELECT 1 FROM threads WHERE hash = $1 FOR UPDATE)"
    )
    .bind(&thread_hash)
    .fetch_one(&mut *tx)
    .await?;

    if !exists {
        return Err(AppError::NotFound("Target thread does not exist".into()));
    }

    let tagging_json = serde_json::to_value(&tags).unwrap();
    let tagged_by_json = serde_json::json!([]);

    sqlx::query(
        r#"
        INSERT INTO posts (hash, thread_hash, topic, name, options, password_hash, comment, file_name, file_mime, file_size, file_dimensions, timestamp, tagging, tagged_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        "#
    )
    .bind(&hash)
    .bind(&thread_hash)
    .bind(&clean_topic)
    .bind(&final_name)
    .bind(&options)
    .bind(&pass_hash)
    .bind(&comment)
    .bind(&file_details.name)
    .bind(&file_details.mime)
    .bind(&file_details.size)
    .bind(&file_details.dimensions)
    .bind(&timestamp)
    .bind(&tagging_json)
    .bind(&tagged_by_json)
    .execute(&mut *tx)
    .await?;

    if !tags.is_empty() {
        let tag_val = serde_json::json!([hash]);
        sqlx::query(
            r#"
            UPDATE posts
            SET tagged_by = COALESCE(tagged_by, '[]'::jsonb) || $1::jsonb
            WHERE hash = ANY($2) AND NOT (COALESCE(tagged_by, '[]'::jsonb) @> $1::jsonb)
            "#
        )
        .bind(&tag_val)
        .bind(&tags)
        .execute(&mut *tx)
        .await?;
    }

    if !options.to_lowercase().contains("sage") {
        sqlx::query("UPDATE threads SET bumped_at = CURRENT_TIMESTAMP WHERE hash = $1")
            .bind(&thread_hash)
            .execute(&mut *tx)
            .await?;
    }

    tx.commit().await?;

    let mut reply_json = serde_json::json!({
        "hash": hash,
        "thread": thread_hash,
        "topic": clean_topic,
        "name": final_name,
        "options": options,
        "comment": comment,
        "file_name": file_details.name,
        "file_mime": file_details.mime,
        "file_size": file_details.size,
        "file_dimensions": file_details.dimensions,
        "timestamp": timestamp,
        "taggedBy": [],
        "tagging": tags
    });

    match state.render_template("reply-item", &reply_json) {
        Ok(rendered) => {
            reply_json["html"] = serde_json::Value::String(rendered);
        }
        Err(e) => {
            tracing::error!("Failed to pre-render reply-item for SSE: {e:#?}");
        }
    }

    state.sse_hub.broadcast(&state.db, &clean_topic, "new-reply", reply_json.clone()).await;

    Ok((StatusCode::CREATED, Json(reply_json)))
}
