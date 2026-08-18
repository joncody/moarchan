use tokio::sync::broadcast;
use sqlx::postgres::PgListener;
use serde_json::Value;
use crate::{models::sse::SseEventEnvelope, state::AppState};

pub struct SseHub {
    tx: broadcast::Sender<SseEventEnvelope>,
}

impl SseHub {
    pub fn new() -> Self {
        let (tx, _) = broadcast::channel(1024);
        Self { tx }
    }

    pub fn subscribe(&self) -> broadcast::Receiver<SseEventEnvelope> {
        self.tx.subscribe()
    }

    pub async fn broadcast(&self, pool: &sqlx::PgPool, topic: &str, event: &str, payload: Value) {
        let envelope = SseEventEnvelope {
            topic: topic.to_string(),
            event: event.to_string(),
            data: payload,
        };

        // Notify cluster via PostgreSQL pg_notify
        if let Ok(serialized) = serde_json::to_string(&envelope) {
            if serialized.len() <= 7500 {
                let _ = sqlx::query("SELECT pg_notify('moarchan_events', $1)")
                    .bind(serialized)
                    .execute(pool)
                    .await;
            } else {
                // Large payload hydration fallback
                let compact = serde_json::json!({
                    "topic": topic,
                    "event": event,
                    "fetch": true,
                    "hash": envelope.data.get("hash").and_then(|h| h.as_str()).unwrap_or("")
                });
                if let Ok(c_str) = serde_json::to_string(&compact) {
                    let _ = sqlx::query("SELECT pg_notify('moarchan_events', $1)")
                        .bind(c_str)
                        .execute(pool)
                        .await;
                }
            }
        }

        // Local in-process fanout
        let _ = self.tx.send(envelope);
    }

    pub fn start_postgres_listener(state: AppState) {
        tokio::spawn(async move {
            let mut listener = match PgListener::connect_with(&state.db).await {
                Ok(l) => l,
                Err(e) => {
                    tracing::error!("Failed to initialize PgListener: {e}");
                    return;
                }
            };

            if let Err(e) = listener.listen("moarchan_events").await {
                tracing::error!("Failed to subscribe to channel moarchan_events: {e}");
                return;
            }

            tracing::info!("PostgreSQL LISTEN moarchan_events worker online");

            loop {
                match listener.recv().await {
                    Ok(notification) => {
                        let payload_str = notification.payload();
                        if let Ok(val) = serde_json::from_str::<Value>(payload_str) {
                            if let (Some(topic), Some(event)) = (
                                val.get("topic").and_then(|t| t.as_str()),
                                val.get("event").and_then(|e| e.as_str()),
                            ) {
                                let mut final_data = val.get("data").cloned().unwrap_or(val.clone());
                                
                                // Hydrate if compact notification was sent
                                if val.get("fetch").and_then(|f| f.as_bool()).unwrap_or(false) {
                                    if let Some(hash) = val.get("hash").and_then(|h| h.as_str()) {
                                        if let Ok(Some(thread)) = crate::db::queries::get_single_thread(&state.db, topic, hash).await {
                                            if let Ok(json_val) = serde_json::to_value(&thread) {
                                                final_data = json_val;
                                            }
                                        }
                                    }
                                }

                                let _ = state.sse_hub.tx.send(SseEventEnvelope {
                                    topic: topic.to_string(),
                                    event: event.to_string(),
                                    data: final_data,
                                });
                            }
                        }
                    }
                    Err(e) => {
                        tracing::warn!("PgListener connection lost: {e}, reconnecting in 5s...");
                        tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
                    }
                }
            }
        });
    }
}
