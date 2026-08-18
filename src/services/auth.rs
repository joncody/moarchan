use crate::error::AppError;

const DUMMY_HASH: &str = "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUU123456";

// Fast hashing for ephemeral deletion passwords (cost 4 = ~1.5ms)
pub async fn hash_password(pwd: &str) -> Result<String, AppError> {
    if pwd.trim().is_empty() {
        return Ok(String::new());
    }
    let p = pwd.to_string();
    tokio::task::spawn_blocking(move || {
        bcrypt::hash(&p, 4).map_err(|e| AppError::Internal(anyhow::anyhow!(e)))
    })
    .await
    .map_err(|e| AppError::Internal(anyhow::anyhow!(e)))?
}

// Standard hashing for registered user accounts (/register)
pub async fn hash_account_password(pwd: &str) -> Result<String, AppError> {
    if pwd.trim().is_empty() {
        return Ok(String::new());
    }
    let p = pwd.to_string();
    tokio::task::spawn_blocking(move || {
        bcrypt::hash(&p, bcrypt::DEFAULT_COST).map_err(|e| AppError::Internal(anyhow::anyhow!(e)))
    })
    .await
    .map_err(|e| AppError::Internal(anyhow::anyhow!(e)))?
}

pub async fn verify_password(hash: Option<&str>, pwd: &str) -> Result<bool, AppError> {
    let target = hash.unwrap_or(DUMMY_HASH).to_string();
    let p = pwd.to_string();

    tokio::task::spawn_blocking(move || {
        bcrypt::verify(&p, &target).unwrap_or(false)
    })
    .await
    .map_err(|e| AppError::Internal(anyhow::anyhow!(e)))
}
