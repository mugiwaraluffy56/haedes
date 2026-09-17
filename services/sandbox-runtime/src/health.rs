use std::sync::Arc;

use axum::{extract::State, Json};
use serde::Serialize;

use crate::config::Config;

#[derive(Debug, Serialize)]
pub struct HealthResponse {
    pub status: &'static str,
    pub version: String,
    #[serde(rename = "sandboxId")]
    pub sandbox_id: String,
    pub workspace: String,
    #[serde(rename = "workspaceAvailable")]
    pub workspace_available: bool,
}

pub async fn health(State(config): State<Arc<Config>>) -> Json<HealthResponse> {
    health_response(&config).await
}

pub async fn runtime_status(State(config): State<Arc<Config>>) -> Json<HealthResponse> {
    health_response(&config).await
}

async fn health_response(config: &Config) -> Json<HealthResponse> {
    let workspace_available = tokio::fs::metadata(config.workspace())
        .await
        .map(|metadata| metadata.is_dir())
        .unwrap_or(false);

    Json(HealthResponse {
        status: "ok",
        version: config.version().to_owned(),
        sandbox_id: config.sandbox_id().to_owned(),
        workspace: config.workspace().display().to_string(),
        workspace_available,
    })
}
