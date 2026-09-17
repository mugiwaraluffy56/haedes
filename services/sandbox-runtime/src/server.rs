use std::sync::Arc;

use axum::{
    extract::State,
    http::{header::AUTHORIZATION, Request},
    middleware::{self, Next},
    response::Response,
    routing::get,
    Router,
};
use subtle::ConstantTimeEq;

use crate::{
    config::Config,
    error::unauthorized,
    health::{health, runtime_status},
};

pub fn build_router(config: Config) -> Router {
    let state = Arc::new(config);
    let protected = Router::new()
        .route("/status", get(runtime_status))
        .layer(middleware::from_fn_with_state(state.clone(), authorize));

    Router::new()
        .route("/healthz", get(health))
        .nest("/v1", protected)
        .with_state(state)
}

async fn authorize(
    State(config): State<Arc<Config>>,
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
    if token.as_bytes().ct_eq(config.runtime_token()).unwrap_u8() != 1 {
        return unauthorized();
    }

    next.run(request).await
}
