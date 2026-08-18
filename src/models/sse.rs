use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SseEventEnvelope {
    pub topic: String,
    pub event: String,
    pub data: serde_json::Value,
}
