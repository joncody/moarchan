use axum::{
    extract::{Query, State},
    response::sse::{Event, KeepAlive, Sse},
};
use futures_util::StreamExt;
use serde::Deserialize;
use std::{convert::Infallible, time::Duration};
use tokio_stream::wrappers::BroadcastStream;
use crate::{error::AppError, state::AppState};

#[derive(Deserialize)]
pub struct StreamQuery {
    pub topic: String,
}

pub async fn sse_stream_handler(
    State(state): State<AppState>,
    Query(query): Query<StreamQuery>,
) -> Result<Sse<impl futures_util::Stream<Item = Result<Event, Infallible>>>, AppError> {
    let target_topic = query.topic.trim_matches('/').to_string();
    let rx = state.sse_hub.subscribe();

    let stream = BroadcastStream::new(rx).filter_map(move |item| {
        let topic = target_topic.clone();
        async move {
            match item {
                Ok(msg) => {
                    let msg_topic = msg.topic.trim_matches('/');
                    if msg_topic == topic || topic.is_empty() {
                        let data_str = serde_json::to_string(&msg.data).unwrap_or_default();
                        Some(Ok(Event::default().event(msg.event).data(data_str)))
                    } else {
                        None
                    }
                }
                _ => None,
            }
        }
    });

    Ok(Sse::new(stream).keep_alive(
        KeepAlive::new()
            .interval(Duration::from_secs(15))
            .text("ping"),
    ))
}
