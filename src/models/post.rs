use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Reply {
    pub hash: String,
    pub thread: String,
    pub topic: String,
    pub name: String,
    pub options: String,
    pub comment: String,
    pub file_name: String,
    pub file_mime: String,
    pub file_size: String,
    pub file_dimensions: String,
    pub timestamp: String,
    #[sqlx(json)]
    pub tagging: Vec<String>,
    #[serde(rename = "taggedBy")]
    #[sqlx(json)]
    pub tagged_by: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Thread {
    pub hash: String,
    pub topic: String,
    pub name: String,
    pub subject: String,
    pub options: String,
    pub comment: String,
    pub file_name: String,
    pub file_mime: String,
    pub file_size: String,
    pub file_dimensions: String,
    pub timestamp: String,
    pub replies: Vec<Reply>,
    #[serde(rename = "taggedBy")]
    pub tagged_by: Vec<String>,
    pub tagging: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileDetails {
    pub name: String,
    pub path: String,
    pub mime: String,
    pub size: String,
    pub dimensions: String,
}

impl Default for FileDetails {
    fn default() -> Self {
        Self {
            name: String::new(),
            path: String::new(),
            mime: String::new(),
            size: String::new(),
            dimensions: String::new(),
        }
    }
}

#[allow(dead_code)]
pub struct DeletedPostInfo {
    pub hash: String,
    pub topic: String,
    pub is_thread: bool,
    pub file_names: Vec<String>,
}
