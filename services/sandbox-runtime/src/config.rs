use std::{env, net::SocketAddr, path::PathBuf, str::FromStr};

use nix::unistd::geteuid;
use thiserror::Error;

#[derive(Clone)]
pub struct Config {
    bind_addr: SocketAddr,
    runtime_token: String,
    sandbox_id: String,
    version: String,
    workspace: PathBuf,
}

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("missing required runtime configuration: {0}")]
    Missing(&'static str),
    #[error("invalid runtime configuration for {field}: {reason}")]
    Invalid { field: &'static str, reason: String },
    #[error("runtime must not run as root")]
    RootProcess,
    #[error("workspace is unavailable: {0}")]
    WorkspaceUnavailable(String),
}

impl Config {
    pub fn from_env() -> Result<Self, ConfigError> {
        Self::from_values(
            env::var("HAEDES_RUNTIME_BIND").unwrap_or_else(|_| "0.0.0.0:8080".to_owned()),
            env::var("HAEDES_RUNTIME_TOKEN")
                .map_err(|_| ConfigError::Missing("HAEDES_RUNTIME_TOKEN"))?,
            env::var("HAEDES_SANDBOX_ID").map_err(|_| ConfigError::Missing("HAEDES_SANDBOX_ID"))?,
            env::var("HAEDES_RUNTIME_VERSION")
                .unwrap_or_else(|_| env!("CARGO_PKG_VERSION").to_owned()),
            env::var("HAEDES_RUNTIME_WORKSPACE")
                .map(PathBuf::from)
                .unwrap_or_else(|_| PathBuf::from("/workspace")),
        )
    }

    pub fn from_values(
        bind_addr: impl AsRef<str>,
        runtime_token: impl Into<String>,
        sandbox_id: impl Into<String>,
        version: impl Into<String>,
        workspace: PathBuf,
    ) -> Result<Self, ConfigError> {
        let bind_addr =
            SocketAddr::from_str(bind_addr.as_ref()).map_err(|error| ConfigError::Invalid {
                field: "HAEDES_RUNTIME_BIND",
                reason: error.to_string(),
            })?;
        let runtime_token = runtime_token.into();
        if runtime_token.is_empty() || runtime_token.len() > 4096 {
            return Err(ConfigError::Invalid {
                field: "HAEDES_RUNTIME_TOKEN",
                reason: "must contain between 1 and 4096 bytes".to_owned(),
            });
        }
        let sandbox_id = sandbox_id.into();
        if !sandbox_id.starts_with("sbx_") || sandbox_id.len() <= 4 {
            return Err(ConfigError::Invalid {
                field: "HAEDES_SANDBOX_ID",
                reason: "must use the sbx_ prefix".to_owned(),
            });
        }
        let version = version.into();
        if version.is_empty() || version.len() > 64 {
            return Err(ConfigError::Invalid {
                field: "HAEDES_RUNTIME_VERSION",
                reason: "must contain between 1 and 64 bytes".to_owned(),
            });
        }
        if !workspace.is_absolute() {
            return Err(ConfigError::Invalid {
                field: "HAEDES_RUNTIME_WORKSPACE",
                reason: "must be an absolute path".to_owned(),
            });
        }

        Ok(Self {
            bind_addr,
            runtime_token,
            sandbox_id,
            version,
            workspace,
        })
    }

    pub fn bind_addr(&self) -> SocketAddr {
        self.bind_addr
    }

    pub(crate) fn runtime_token(&self) -> &[u8] {
        self.runtime_token.as_bytes()
    }

    pub(crate) fn sandbox_id(&self) -> &str {
        &self.sandbox_id
    }

    pub(crate) fn version(&self) -> &str {
        &self.version
    }

    pub(crate) fn workspace(&self) -> &PathBuf {
        &self.workspace
    }

    pub fn ensure_non_root(&self) -> Result<(), ConfigError> {
        if geteuid().is_root() {
            return Err(ConfigError::RootProcess);
        }
        Ok(())
    }

    pub async fn ensure_workspace(&self) -> Result<(), ConfigError> {
        let metadata = tokio::fs::metadata(&self.workspace)
            .await
            .map_err(|error| ConfigError::WorkspaceUnavailable(error.to_string()))?;
        if !metadata.is_dir() {
            return Err(ConfigError::WorkspaceUnavailable(
                "path is not a directory".to_owned(),
            ));
        }
        Ok(())
    }
}
