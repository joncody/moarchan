use sqlx::{PgPool, Row};
use serde_json::Value;
use crate::{
    error::AppError,
    models::post::{DeletedPostInfo, Reply, Thread},
    services::auth::verify_password,
};

pub async fn get_topic_threads(pool: &PgPool, topic: &str) -> Result<Vec<Thread>, AppError> {
    let rows = sqlx::query(
        r#"
        SELECT
            t.hash, t.topic, t.name, COALESCE(t.subject, '') as subject, COALESCE(t.options, '') as options, t.comment,
            t.file_name, t.file_mime, t.file_size, t.file_dimensions, t.timestamp,
            COALESCE(
                jsonb_agg(
                    jsonb_build_object(
                        'hash', p.hash,
                        'thread', p.thread_hash,
                        'topic', p.topic,
                        'name', p.name,
                        'options', COALESCE(p.options, ''),
                        'comment', p.comment,
                        'file_name', COALESCE(p.file_name, ''),
                        'file_mime', COALESCE(p.file_mime, ''),
                        'file_size', COALESCE(p.file_size, ''),
                        'file_dimensions', COALESCE(p.file_dimensions, ''),
                        'timestamp', p.timestamp,
                        'tagging', p.tagging,
                        'taggedBy', p.tagged_by
                    ) ORDER BY p.id ASC
                ) FILTER (WHERE p.hash IS NOT NULL),
                '[]'::jsonb
            ) as replies
        FROM threads t
        LEFT JOIN posts p ON t.hash = p.thread_hash
        WHERE t.topic = $1
        GROUP BY t.id, t.hash, t.bumped_at
        ORDER BY t.bumped_at DESC
        LIMIT 100
        "#
    )
    .bind(topic)
    .fetch_all(pool)
    .await?;

    let mut threads = Vec::with_capacity(rows.len());
    for r in rows {
        let replies_json: Value = r.try_get("replies")?;
        let replies: Vec<Reply> = serde_json::from_value(replies_json).unwrap_or_default();

        threads.push(Thread {
            hash: r.try_get("hash")?,
            topic: r.try_get("topic")?,
            name: r.try_get("name")?,
            subject: r.try_get("subject")?,
            options: r.try_get("options")?,
            comment: r.try_get("comment")?,
            file_name: r.try_get("file_name")?,
            file_mime: r.try_get("file_mime")?,
            file_size: r.try_get("file_size")?,
            file_dimensions: r.try_get("file_dimensions")?,
            timestamp: r.try_get("timestamp")?,
            replies,
            tagged_by: Vec::new(),
            tagging: Vec::new(),
        });
    }

    Ok(threads)
}

pub async fn get_single_thread(pool: &PgPool, topic: &str, hash: &str) -> Result<Option<Thread>, AppError> {
    let row = sqlx::query(
        r#"
        SELECT
            t.hash, t.topic, t.name, COALESCE(t.subject, '') as subject, COALESCE(t.options, '') as options, t.comment,
            t.file_name, t.file_mime, t.file_size, t.file_dimensions, t.timestamp,
            COALESCE(
                jsonb_agg(
                    jsonb_build_object(
                        'hash', p.hash,
                        'thread', p.thread_hash,
                        'topic', p.topic,
                        'name', p.name,
                        'options', COALESCE(p.options, ''),
                        'comment', p.comment,
                        'file_name', COALESCE(p.file_name, ''),
                        'file_mime', COALESCE(p.file_mime, ''),
                        'file_size', COALESCE(p.file_size, ''),
                        'file_dimensions', COALESCE(p.file_dimensions, ''),
                        'timestamp', p.timestamp,
                        'tagging', p.tagging,
                        'taggedBy', p.tagged_by
                    ) ORDER BY p.id ASC
                ) FILTER (WHERE p.hash IS NOT NULL),
                '[]'::jsonb
            ) as replies
        FROM threads t
        LEFT JOIN posts p ON t.hash = p.thread_hash
        WHERE t.topic = $1 AND t.hash = $2
        GROUP BY t.id, t.hash
        "#
    )
    .bind(topic)
    .bind(hash)
    .fetch_optional(pool)
    .await?;

    if let Some(r) = row {
        let replies_json: Value = r.try_get("replies")?;
        let replies: Vec<Reply> = serde_json::from_value(replies_json).unwrap_or_default();

        Ok(Some(Thread {
            hash: r.try_get("hash")?,
            topic: r.try_get("topic")?,
            name: r.try_get("name")?,
            subject: r.try_get("subject")?,
            options: r.try_get("options")?,
            comment: r.try_get("comment")?,
            file_name: r.try_get("file_name")?,
            file_mime: r.try_get("file_mime")?,
            file_size: r.try_get("file_size")?,
            file_dimensions: r.try_get("file_dimensions")?,
            timestamp: r.try_get("timestamp")?,
            replies,
            tagged_by: Vec::new(),
            tagging: Vec::new(),
        }))
    } else {
        Ok(None)
    }
}

pub async fn delete_post_or_file(
    pool: &PgPool,
    hash: &str,
    password: &str,
    is_admin: bool,
    file_only: bool,
) -> Result<DeletedPostInfo, AppError> {
    let mut tx = pool.begin().await?;

    let thread_record = sqlx::query(
        "SELECT topic, COALESCE(password_hash, '') as pass_hash, COALESCE(file_name, '') as file_name FROM threads WHERE hash = $1 FOR UPDATE"
    )
    .bind(hash)
    .fetch_optional(&mut *tx)
    .await?;

    let (topic, db_pass, file_name, is_thread) = if let Some(t) = thread_record {
        let topic: String = t.try_get("topic")?;
        let pass_hash: String = t.try_get("pass_hash")?;
        let file_name: String = t.try_get("file_name")?;
        (topic, pass_hash, file_name, true)
    } else {
        let post_record = sqlx::query(
            "SELECT topic, COALESCE(password_hash, '') as pass_hash, COALESCE(file_name, '') as file_name FROM posts WHERE hash = $1 FOR UPDATE"
        )
        .bind(hash)
        .fetch_optional(&mut *tx)
        .await?
        .ok_or_else(|| AppError::NotFound(format!("Post or thread '{hash}' not found")))?;

        let topic: String = post_record.try_get("topic")?;
        let pass_hash: String = post_record.try_get("pass_hash")?;
        let file_name: String = post_record.try_get("file_name")?;
        (topic, pass_hash, file_name, false)
    };

    if !is_admin {
        if db_pass.is_empty() {
            return Err(AppError::Forbidden("Post cannot be deleted without admin privileges".into()));
        }
        let valid = verify_password(Some(&db_pass), password).await?;
        if !valid {
            return Err(AppError::Forbidden("Invalid post deletion password".into()));
        }
    }

    let mut file_names = Vec::new();
    if !file_name.is_empty() {
        file_names.push(file_name);
    }

    if file_only {
        if is_thread {
            sqlx::query("UPDATE threads SET file_name = '', file_mime = '', file_size = '', file_dimensions = '' WHERE hash = $1")
                .bind(hash)
                .execute(&mut *tx)
                .await?;
        } else {
            sqlx::query("UPDATE posts SET file_name = '', file_mime = '', file_size = '', file_dimensions = '' WHERE hash = $1")
                .bind(hash)
                .execute(&mut *tx)
                .await?;
        }
    } else if is_thread {
        let reply_files = sqlx::query("SELECT file_name FROM posts WHERE thread_hash = $1 AND file_name != ''")
            .bind(hash)
            .fetch_all(&mut *tx)
            .await?;
        for rf in reply_files {
            let rfn: String = rf.try_get("file_name")?;
            if !rfn.is_empty() {
                file_names.push(rfn);
            }
        }
        sqlx::query("DELETE FROM threads WHERE hash = $1")
            .bind(hash)
            .execute(&mut *tx)
            .await?;
    } else {
        sqlx::query("DELETE FROM posts WHERE hash = $1")
            .bind(hash)
            .execute(&mut *tx)
            .await?;
    }

    tx.commit().await?;

    Ok(DeletedPostInfo {
        hash: hash.to_string(),
        topic,
        is_thread,
        file_names,
    })
}
