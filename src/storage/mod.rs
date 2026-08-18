pub mod local;

use async_trait::async_trait;
use crate::error::AppError;

#[async_trait]
pub trait StorageBackend: Send + Sync {
    async fn save(&self, rel_path: &str, data: &[u8]) -> Result<(), AppError>;
    async fn delete(&self, rel_path: &str) -> Result<(), AppError>;
    fn public_url(&self, rel_path: &str) -> String;
}
