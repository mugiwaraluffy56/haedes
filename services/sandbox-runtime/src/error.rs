use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::Serialize;

#[derive(Debug, Serialize)]
pub struct ErrorEnvelope {
    pub error: ErrorPayload,
}

#[derive(Debug, Serialize)]
pub struct ErrorPayload {
    pub code: &'static str,
    pub message: &'static str,
    #[serde(rename = "requestId")]
    pub request_id: &'static str,
    pub details: serde_json::Value,
}

pub fn unauthorized() -> Response {
    (
        StatusCode::UNAUTHORIZED,
        Json(ErrorEnvelope {
            error: ErrorPayload {
                code: "runtime_unauthorized",
                message: "A valid sandbox runtime token is required.",
                request_id: "req_runtime_auth",
                details: serde_json::json!({}),
            },
        }),
    )
        .into_response()
}
