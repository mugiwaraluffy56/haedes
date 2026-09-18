use std::{collections::HashMap, convert::Infallible, sync::Arc, time::Duration};

use async_stream::stream;
use axum::{
    body::Bytes,
    extract::{Path, Query, State},
    http::{header::AUTHORIZATION, HeaderMap, Request, StatusCode},
    middleware::{self, Next},
    response::{
        sse::{Event, KeepAlive, Sse},
        IntoResponse, Response,
    },
    routing::{get, post, put},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use subtle::ConstantTimeEq;

use crate::{
    config::Config,
    error::{unauthorized, ErrorEnvelope, ErrorPayload},
    filesystem::{FileService, PathGuard},
    health::{health, runtime_status},
    process::{CommandRequest, CommandRunner},
    snapshot::{ArchiveService, MAX_ARCHIVE_BYTES},
    streaming::CommandEventKind,
};

#[derive(Clone)]
pub struct RuntimeState {
    pub(crate) config: Config,
    pub(crate) runner: CommandRunner,
    pub(crate) files: FileService,
    pub(crate) archives: ArchiveService,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct CommandPayload {
    command: String,
    cwd: Option<String>,
    #[serde(default)]
    environment: HashMap<String, String>,
    timeout_seconds: u64,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct CommandResponse {
    id: String,
    sandbox_id: String,
    command: String,
    exit_code: Option<i32>,
    signal: Option<String>,
    timed_out: bool,
}

#[derive(Debug, Deserialize)]
struct PathQuery {
    path: Option<String>,
}

pub fn build_router(config: Config) -> Router {
    // main validates the workspace before serving traffic. The fallback keeps
    // health-only test routers constructible so they can report an unhealthy
    // workspace instead of panicking during setup.
    let guard = PathGuard::new(config.workspace()).unwrap_or_else(|_| {
        PathGuard::new(std::env::temp_dir()).expect("system temporary directory")
    });
    let state = Arc::new(RuntimeState {
        config,
        runner: CommandRunner::new(guard.clone()),
        files: FileService::new(guard.clone()),
        archives: ArchiveService::new(guard),
    });
    let protected = Router::new()
        .route("/status", get(runtime_status))
        .route("/commands", post(start_command))
        .route("/commands/:id/events", get(command_events))
        .route("/files", get(list_files))
        .route(
            "/files/content",
            get(read_file).put(write_file).delete(delete_file),
        )
        .route("/snapshot/export", get(export_workspace))
        .route("/snapshot/restore", put(restore_workspace))
        .fallback(|| async { StatusCode::NOT_FOUND })
        .layer(middleware::from_fn_with_state(state.clone(), authorize));

    Router::new()
        .route("/healthz", get(health))
        .nest("/v1", protected)
        .with_state(state)
}

async fn authorize(
    State(state): State<Arc<RuntimeState>>,
    request: Request<axum::body::Body>,
    next: Next,
) -> Response {
    let Some(header) = request.headers().get(AUTHORIZATION) else {
        return unauthorized();
    };
    let Ok(header) = header.to_str() else {
        return unauthorized();
    };
    let Some(token) = header.strip_prefix("Bearer ") else {
        return unauthorized();
    };
    if token
        .as_bytes()
        .ct_eq(state.config.runtime_token())
        .unwrap_u8()
        != 1
    {
        return unauthorized();
    }

    next.run(request).await
}

async fn start_command(
    State(state): State<Arc<RuntimeState>>,
    Json(payload): Json<CommandPayload>,
) -> Response {
    if payload.timeout_seconds == 0 || payload.timeout_seconds > 900 {
        return runtime_error(
            StatusCode::BAD_REQUEST,
            "command_timeout",
            "timeoutSeconds must be between 1 and 900",
        );
    }
    let command = payload.command.clone();
    let handle = match state.runner.start(CommandRequest {
        command: payload.command,
        cwd: payload.cwd,
        environment: payload.environment,
        timeout: Duration::from_secs(payload.timeout_seconds),
    }) {
        Ok(handle) => handle,
        Err(error) => {
            return runtime_error(StatusCode::BAD_REQUEST, error.code(), &error.to_string())
        }
    };
    let mut receiver = match handle.events(0) {
        Ok(receiver) => receiver,
        Err(error) => {
            return runtime_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                "runtime_unavailable",
                &error.to_string(),
            )
        }
    };
    loop {
        match receiver.recv().await {
            Ok(Some(event)) => match event.kind {
                CommandEventKind::Completed(summary) => {
                    return (
                        StatusCode::OK,
                        Json(CommandResponse {
                            id: handle.id().to_owned(),
                            sandbox_id: state.config.sandbox_id().to_owned(),
                            command,
                            exit_code: summary.exit_code,
                            signal: summary.signal,
                            timed_out: summary.timed_out,
                        }),
                    )
                        .into_response();
                }
                CommandEventKind::Failed { code, message } => {
                    return runtime_error(StatusCode::INTERNAL_SERVER_ERROR, &code, &message);
                }
                _ => {}
            },
            Ok(None) => {
                return runtime_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "runtime_unavailable",
                    "command ended without a result",
                )
            }
            Err(error) => {
                return runtime_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "runtime_unavailable",
                    &error.to_string(),
                )
            }
        }
    }
}

async fn command_events(
    State(state): State<Arc<RuntimeState>>,
    Path(id): Path<String>,
    headers: HeaderMap,
) -> Response {
    let after = headers
        .get("last-event-id")
        .and_then(|value| value.to_str().ok())
        .and_then(|value| value.parse::<u64>().ok())
        .unwrap_or(0);
    let mut receiver = match state.runner.events(&id, after) {
        Ok(receiver) => receiver,
        Err(error) => {
            return runtime_error(
                StatusCode::NOT_FOUND,
                "command_not_found",
                &error.to_string(),
            )
        }
    };
    let stream = stream! {
        loop {
            match receiver.recv().await {
                Ok(Some(event)) => {
                    let terminal = matches!(event.kind, CommandEventKind::Completed(_) | CommandEventKind::Failed { .. });
                    let (kind, payload) = event_payload(&event.kind);
                    let sse_event = Event::default()
                        .id(event.sequence.to_string())
                        .event(kind)
                        .json_data(payload)
                        .unwrap_or_else(|_| Event::default().event("failed"));
                    yield Ok::<Event, Infallible>(sse_event);
                    if terminal { break; }
                }
                Ok(None) | Err(_) => break,
            }
        }
    };
    Sse::new(stream)
        .keep_alive(KeepAlive::new().interval(Duration::from_secs(15)))
        .into_response()
}

fn event_payload(kind: &CommandEventKind) -> (&'static str, serde_json::Value) {
    match kind {
        CommandEventKind::Started => ("started", serde_json::json!({})),
        CommandEventKind::Stdout(data) => (
            "stdout",
            serde_json::json!({"data": String::from_utf8_lossy(data)}),
        ),
        CommandEventKind::Stderr(data) => (
            "stderr",
            serde_json::json!({"data": String::from_utf8_lossy(data)}),
        ),
        CommandEventKind::Completed(summary) => (
            "completed",
            serde_json::json!({
                "result": {"exitCode": summary.exit_code, "signal": summary.signal, "timedOut": summary.timed_out}
            }),
        ),
        CommandEventKind::Failed { code, message } => (
            "failed",
            serde_json::json!({"code": code, "message": message}),
        ),
    }
}

async fn list_files(
    State(state): State<Arc<RuntimeState>>,
    Query(query): Query<PathQuery>,
) -> Response {
    match state
        .files
        .list(query.path.as_deref().unwrap_or("/workspace"))
        .await
    {
        Ok(entries) => Json(entries).into_response(),
        Err(error) => runtime_error(StatusCode::NOT_FOUND, error.code(), &error.to_string()),
    }
}

async fn read_file(
    State(state): State<Arc<RuntimeState>>,
    Query(query): Query<PathQuery>,
) -> Response {
    let Some(path) = query.path else {
        return runtime_error(
            StatusCode::BAD_REQUEST,
            "invalid_request",
            "path is required",
        );
    };
    match state.files.read(&path).await {
        Ok(body) => (StatusCode::OK, body).into_response(),
        Err(error) => runtime_error(StatusCode::NOT_FOUND, error.code(), &error.to_string()),
    }
}

async fn write_file(
    State(state): State<Arc<RuntimeState>>,
    Query(query): Query<PathQuery>,
    body: Bytes,
) -> Response {
    let Some(path) = query.path else {
        return runtime_error(
            StatusCode::BAD_REQUEST,
            "invalid_request",
            "path is required",
        );
    };
    match state.files.write(&path, &body).await {
        Ok(entry) => Json(entry).into_response(),
        Err(error) => runtime_error(StatusCode::BAD_REQUEST, error.code(), &error.to_string()),
    }
}

async fn delete_file(
    State(state): State<Arc<RuntimeState>>,
    Query(query): Query<PathQuery>,
) -> Response {
    let Some(path) = query.path else {
        return runtime_error(
            StatusCode::BAD_REQUEST,
            "invalid_request",
            "path is required",
        );
    };
    match state.files.delete(&path).await {
        Ok(()) => StatusCode::NO_CONTENT.into_response(),
        Err(error) => runtime_error(StatusCode::NOT_FOUND, error.code(), &error.to_string()),
    }
}

async fn export_workspace(State(state): State<Arc<RuntimeState>>) -> Response {
    match state.archives.export() {
        Ok((archive, info)) => {
            let mut headers = HeaderMap::new();
            headers.insert("content-type", info.media_type.parse().unwrap());
            headers.insert(
                "content-length",
                info.byte_size.to_string().parse().unwrap(),
            );
            headers.insert("x-haedes-sha256", info.sha256.parse().unwrap());
            (headers, archive).into_response()
        }
        Err(error) => runtime_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            "snapshot_export_failed",
            &error.to_string(),
        ),
    }
}

async fn restore_workspace(
    State(state): State<Arc<RuntimeState>>,
    headers: HeaderMap,
    body: Bytes,
) -> Response {
    if body.len() > MAX_ARCHIVE_BYTES {
        return runtime_error(
            StatusCode::PAYLOAD_TOO_LARGE,
            "archive_too_large",
            "archive exceeds the configured limit",
        );
    }
    let checksum = headers
        .get("x-haedes-sha256")
        .and_then(|value| value.to_str().ok());
    match state.archives.restore(&body, checksum) {
        Ok(info) => Json(info).into_response(),
        Err(error) => runtime_error(
            StatusCode::BAD_REQUEST,
            "snapshot_restore_failed",
            &error.to_string(),
        ),
    }
}

fn runtime_error(status: StatusCode, code: &str, message: &str) -> Response {
    (
        status,
        Json(ErrorEnvelope {
            error: ErrorPayload {
                code: "runtime_failure",
                message: "The sandbox runtime could not complete the request.",
                request_id: "req_runtime",
                details: serde_json::json!({"code": code, "message": message}),
            },
        }),
    )
        .into_response()
}
