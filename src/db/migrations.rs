use sqlx::{PgPool, Postgres, Transaction};
use crate::error::AppError;

pub async fn run_migrations(pool: &PgPool) -> Result<(), AppError> {
    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version TEXT PRIMARY KEY,
            description TEXT NOT NULL,
            applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
        );
        "#
    )
    .execute(pool)
    .await?;

    apply_migration(pool, "001_create_boards", "Create boards table", |tx| Box::pin(async move {
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS boards (
                id BIGSERIAL PRIMARY KEY,
                slug TEXT UNIQUE NOT NULL
            );
            "#
        ).execute(&mut **tx).await?;
        Ok(())
    })).await?;

    apply_migration(pool, "002_create_threads", "Create threads table", |tx| Box::pin(async move {
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS threads (
                id BIGSERIAL PRIMARY KEY,
                hash TEXT UNIQUE NOT NULL,
                topic TEXT NOT NULL,
                name TEXT NOT NULL,
                subject TEXT,
                options TEXT,
                password_hash TEXT,
                comment TEXT NOT NULL,
                file_name TEXT NOT NULL,
                file_mime TEXT NOT NULL,
                file_size TEXT NOT NULL,
                file_dimensions TEXT NOT NULL,
                timestamp TEXT NOT NULL,
                bumped_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
                created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
            );
            "#
        ).execute(&mut **tx).await?;
        Ok(())
    })).await?;

    apply_migration(pool, "003_create_posts", "Create posts table", |tx| Box::pin(async move {
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS posts (
                id BIGSERIAL PRIMARY KEY,
                hash TEXT UNIQUE NOT NULL,
                thread_hash TEXT NOT NULL REFERENCES threads(hash) ON DELETE CASCADE,
                topic TEXT NOT NULL,
                name TEXT NOT NULL,
                options TEXT,
                password_hash TEXT,
                comment TEXT NOT NULL,
                file_name TEXT,
                file_mime TEXT,
                file_size TEXT,
                file_dimensions TEXT,
                timestamp TEXT NOT NULL,
                tagging JSONB DEFAULT '[]'::jsonb,
                tagged_by JSONB DEFAULT '[]'::jsonb,
                created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
            );
            "#
        ).execute(&mut **tx).await?;
        Ok(())
    })).await?;

    apply_migration(pool, "004_create_auth", "Create auth store table", |tx| Box::pin(async move {
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS auth (
                id BIGSERIAL PRIMARY KEY,
                key TEXT UNIQUE NOT NULL,
                value JSONB NOT NULL
            );
            "#
        ).execute(&mut **tx).await?;
        Ok(())
    })).await?;

    apply_migration(pool, "005_create_indexes", "Create performance indexes", |tx| Box::pin(async move {
        sqlx::query("CREATE INDEX IF NOT EXISTS idx_threads_topic_bumped ON threads(topic, bumped_at DESC);")
            .execute(&mut **tx).await?;
        sqlx::query("CREATE INDEX IF NOT EXISTS idx_posts_thread_hash ON posts(thread_hash);")
            .execute(&mut **tx).await?;
        Ok(())
    })).await?;

    Ok(())
}

async fn apply_migration<F>(
    pool: &PgPool,
    version: &str,
    description: &str,
    migration_fn: F,
) -> Result<(), AppError>
where
    F: for<'c> FnOnce(&'c mut Transaction<'_, Postgres>) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), AppError>> + Send + 'c>>,
{
    let exists: bool = sqlx::query_scalar(
        "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)"
    )
    .bind(version)
    .fetch_one(pool)
    .await?;

    if exists {
        return Ok(());
    }

    let mut tx = pool.begin().await?;
    migration_fn(&mut tx).await?;

    sqlx::query(
        "INSERT INTO schema_migrations (version, description) VALUES ($1, $2)"
    )
    .bind(version)
    .bind(description)
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;
    tracing::info!("Applied schema migration: {} ({})", version, description);
    Ok(())
}
