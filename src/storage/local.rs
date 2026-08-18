use async_trait::async_trait;
use std::path::{Path, PathBuf};
use tokio::fs;
use crate::{error::AppError, storage::StorageBackend};

pub struct LocalDiskStorage {
    base_path: PathBuf,
    url_prefix: String,
}

impl LocalDiskStorage {
    pub async fn new(base_path: &str, url_prefix: &str) -> Result<Self, AppError> {
        let path = Path::new(base_path).to_path_buf();
        fs::create_dir_all(&path).await
            .map_err(|e| AppError::Internal(anyhow::anyhow!("Failed to create storage dir: {e}")))?;

        Ok(Self {
            base_path: path,
            url_prefix: url_prefix.trim_end_matches('/').to_string(),
        })
    }
}

#[async_trait]
impl StorageBackend for LocalDiskStorage {
    async fn save(&self, rel_path: &str, data: &[u8]) -> Result<(), AppError> {
        let full_path = self.base_path.join(rel_path);
        if let Some(parent) = full_path.parent() {
            fs::create_dir_all(parent).await
                .map_err(|e| AppError::Internal(anyhow::anyhow!("Create parent dir: {e}")))?;
        }
        fs::write(&full_path, data).await
            .map_err(|e| AppError::Internal(anyhow::anyhow!("Failed to write file {}: {e}", full_path.display())))?;
        Ok(())
    }

    async fn delete(&self, rel_path: &str) -> Result<(), AppError> {
        let full_path = self.base_path.join(rel_path);
        if full_path.exists() {
            fs::remove_file(&full_path).await
                .map_err(|e| AppError::Internal(anyhow::anyhow!("Failed to delete file {}: {e}", full_path.display())))?;
        }
        Ok(())
    }

    fn public_url(&self, rel_path: &str) -> String {
        format!("{}/{}", self.url_prefix, rel_path.trim_start_matches('/'))
    }
}
