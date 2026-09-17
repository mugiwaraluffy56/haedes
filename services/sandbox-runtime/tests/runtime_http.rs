use axum::{
    body::to_bytes,
    http::{Request, StatusCode},
};
use haedes_sandbox_runtime::{config::Config, server::build_router};
use serde_json::Value;
use tempfile::tempdir;
use tower::ServiceExt;

async fn test_router() -> axum::Router {
    let workspace = tempdir().expect("workspace directory");
    let workspace_path = workspace.keep();
    let config = Config::from_values(
        "127.0.0.1:0",
        "test-runtime-token",
        "sbx_test",
        "test-version",
        workspace_path,
    )
    .expect("test config");
    build_router(config)
}

#[tokio::test]
async fn health_is_public_and_reports_workspace() {
    let response = test_router()
        .await
        .oneshot(
            Request::get("/healthz")
                .body(axum::body::Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let json: Value = serde_json::from_slice(&body).unwrap();
    assert_eq!(json["status"], "ok");
    assert_eq!(json["version"], "test-version");
    assert_eq!(json["sandboxId"], "sbx_test");
    assert_eq!(json["workspaceAvailable"], true);
}

#[tokio::test]
async fn internal_routes_require_the_sandbox_token() {
    let router = test_router().await;
    let response = router
        .clone()
        .oneshot(
            Request::get("/v1/exec")
                .body(axum::body::Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let json: Value = serde_json::from_slice(&body).unwrap();
    assert_eq!(json["error"]["code"], "runtime_unauthorized");
    assert_eq!(json["error"]["requestId"], "req_runtime_auth");

    let response = router
        .clone()
        .oneshot(
            Request::get("/v1/status")
                .header("authorization", "Bearer wrong-token")
                .body(axum::body::Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);

    let response = router
        .oneshot(
            Request::get("/v1/status")
                .header("authorization", "Bearer test-runtime-token")
                .body(axum::body::Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn missing_workspace_is_reported_unhealthy() {
    let parent = tempdir().expect("workspace parent");
    let missing_workspace = parent.path().join("missing");
    let config = Config::from_values(
        "127.0.0.1:0",
        "test-runtime-token",
        "sbx_test",
        "test-version",
        missing_workspace,
    )
    .unwrap();
    let response = build_router(config)
        .oneshot(
            Request::get("/healthz")
                .body(axum::body::Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    let body = to_bytes(response.into_body(), usize::MAX).await.unwrap();
    let json: Value = serde_json::from_slice(&body).unwrap();
    assert_eq!(json["workspaceAvailable"], false);
}
