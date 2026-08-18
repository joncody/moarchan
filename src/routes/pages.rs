use axum::{
    extract::{Query, Request, State},
    response::{Html, IntoResponse},
    Json,
};
use serde::{Deserialize, Serialize};
use crate::{db::queries::{get_single_thread, get_topic_threads}, error::AppError, state::AppState};

#[derive(Deserialize)]
pub struct RenderQuery {
    pub path: Option<String>,
}

#[derive(Serialize)]
pub struct RenderResponse {
    pub template: String,
    pub controllers: Vec<String>,
}

pub async fn base_shell_handler(
    State(state): State<AppState>,
    req: Request,
) -> Result<Html<String>, AppError> {
    let csrf_token = req.extensions().get::<String>().cloned().unwrap_or_default();
    let session = state.extract_session(req.headers());

    let ctx = minijinja::context! {
        csrf_token => csrf_token,
        alias => session.as_ref().map(|s| s.alias.as_str()).unwrap_or(""),
        privilege => session.as_ref().map(|s| s.privilege.as_str()).unwrap_or(""),
    };

    let tmpl = state.get_template("base")?;
    Ok(Html(tmpl.render(ctx)?))
}

pub async fn render_spa_handler(
    State(state): State<AppState>,
    Query(query): Query<RenderQuery>,
    req: Request,
) -> Result<impl IntoResponse, AppError> {
    let path = query.path.unwrap_or_else(|| "/".into());
    let session = state.extract_session(req.headers());
    let csrf_token = req.extensions().get::<String>().cloned().unwrap_or_default();

    let (tmpl_name, ctrls, data) = resolve_route_data(&state, &path).await?;

    let ctx = minijinja::context! {
        csrf_token => csrf_token,
        alias => session.as_ref().map(|s| s.alias.as_str()).unwrap_or(""),
        privilege => session.as_ref().map(|s| s.privilege.as_str()).unwrap_or(""),
        data => data,
        threads => data,
    };

    let tmpl = state.get_template(&tmpl_name)?;
    let rendered_html = tmpl.render(ctx)?;

    Ok(Json(RenderResponse {
        template: rendered_html,
        controllers: ctrls,
    }))
}

async fn resolve_route_data(
    state: &AppState,
    path: &str,
) -> Result<(String, Vec<String>, serde_json::Value), AppError> {
    let segments: Vec<&str> = path.trim_matches('/').split('/').filter(|s| !s.is_empty()).collect();

    match segments.as_slice() {
        [] => Ok(("main".into(), vec!["main".into()], serde_json::json!({}))),
        ["about"] => Ok(("about".into(), vec![], serde_json::json!({}))),
        ["advertise"] => Ok(("advertise".into(), vec![], serde_json::json!({}))),
        ["blog"] => Ok(("blog".into(), vec![], serde_json::json!({}))),
        ["contact"] => Ok(("contact".into(), vec![], serde_json::json!({}))),
        ["feedback"] => Ok(("feedback".into(), vec![], serde_json::json!({}))),
        ["legal"] => Ok(("legal".into(), vec![], serde_json::json!({}))),
        ["news"] => Ok(("news".into(), vec![], serde_json::json!({}))),
        ["faq"] => Ok(("faq".into(), vec![], serde_json::json!({}))),
        ["press"] => Ok(("press".into(), vec![], serde_json::json!({}))),
        ["rules"] => Ok(("rules".into(), vec![], serde_json::json!({}))),
        ["auth"] => Ok(("auth".into(), vec!["auth".into()], serde_json::json!({}))),
        [topic] => {
            let threads = get_topic_threads(&state.db, topic).await?;
            let val = serde_json::to_value(&threads).unwrap_or_default();
            Ok(("topic".into(), vec!["service".into()], val))
        }
        [topic, "thread", hash] => {
            let thread = get_single_thread(&state.db, topic, hash).await?
                .ok_or_else(|| AppError::NotFound(format!("Thread {hash} not found")))?;
            let val = serde_json::to_value(&thread).unwrap_or_default();
            Ok(("thread".into(), vec!["service".into()], val))
        }
        _ => Err(AppError::NotFound(format!("No route matching path '{path}'"))),
    }
}
