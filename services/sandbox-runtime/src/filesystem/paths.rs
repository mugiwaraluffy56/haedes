use std::{
    fs,
    path::{Component, Path, PathBuf},
};

use thiserror::Error;

const VIRTUAL_WORKSPACE: &str = "/workspace";

#[derive(Debug, Error, PartialEq, Eq)]
pub enum PathError {
    #[error("workspace root is unavailable: {0}")]
    WorkspaceUnavailable(String),
    #[error("path contains a NUL byte")]
    NulByte,
    #[error("encoded path segments are not accepted")]
    EncodedPath,
    #[error("path contains traversal components")]
    Traversal,
    #[error("path is outside the workspace")]
    OutsideWorkspace,
    #[error("workspace path has a missing ancestor")]
    MissingAncestor,
    #[error("workspace path contains an unsafe symlink")]
    UnsafeSymlink,
    #[error("workspace path contains an invalid component")]
    InvalidComponent,
    #[error("filesystem error while resolving workspace path: {0}")]
    Filesystem(String),
}

impl PathError {
    pub fn code(&self) -> &'static str {
        match self {
            Self::WorkspaceUnavailable(_) => "workspace_unavailable",
            Self::Filesystem(_) => "runtime_filesystem_error",
            Self::NulByte
            | Self::EncodedPath
            | Self::Traversal
            | Self::OutsideWorkspace
            | Self::MissingAncestor
            | Self::UnsafeSymlink
            | Self::InvalidComponent => "path_outside_workspace",
        }
    }
}

#[derive(Clone, Debug)]
pub struct PathGuard {
    root: PathBuf,
}

impl PathGuard {
    pub fn new(root: impl AsRef<Path>) -> Result<Self, PathError> {
        let root = fs::canonicalize(root.as_ref())
            .map_err(|error| PathError::WorkspaceUnavailable(error.to_string()))?;
        let metadata = fs::metadata(&root)
            .map_err(|error| PathError::WorkspaceUnavailable(error.to_string()))?;
        if !metadata.is_dir() {
            return Err(PathError::WorkspaceUnavailable(
                "workspace root is not a directory".to_owned(),
            ));
        }

        Ok(Self { root })
    }

    pub fn root(&self) -> &Path {
        &self.root
    }

    pub fn resolve(&self, user_path: &str) -> Result<PathBuf, PathError> {
        if user_path.as_bytes().contains(&0) {
            return Err(PathError::NulByte);
        }
        if contains_percent_escape(user_path) {
            return Err(PathError::EncodedPath);
        }

        let relative = self.relative_path(user_path)?;
        let candidate = self.root.join(relative);
        self.resolve_candidate(&candidate)
    }

    fn relative_path(&self, user_path: &str) -> Result<PathBuf, PathError> {
        let path = Path::new(user_path);
        let relative = if path.is_absolute() {
            path.strip_prefix(&self.root)
                .or_else(|_| path.strip_prefix(VIRTUAL_WORKSPACE))
                .map_err(|_| PathError::OutsideWorkspace)?
        } else {
            path
        };

        for component in relative.components() {
            match component {
                Component::CurDir | Component::Normal(_) => {}
                Component::ParentDir => return Err(PathError::Traversal),
                Component::RootDir | Component::Prefix(_) => {
                    return Err(PathError::InvalidComponent)
                }
            }
        }

        Ok(relative.to_path_buf())
    }

    fn resolve_candidate(&self, candidate: &Path) -> Result<PathBuf, PathError> {
        match fs::symlink_metadata(candidate) {
            Ok(metadata) if metadata.file_type().is_symlink() => {
                let resolved = fs::canonicalize(candidate).map_err(|_| PathError::UnsafeSymlink)?;
                if resolved.starts_with(&self.root) {
                    Ok(resolved)
                } else {
                    Err(PathError::UnsafeSymlink)
                }
            }
            Ok(_) => {
                let resolved = fs::canonicalize(candidate)
                    .map_err(|error| PathError::Filesystem(error.to_string()))?;
                self.ensure_inside(&resolved)
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                self.resolve_missing_leaf(candidate)
            }
            Err(error) => Err(PathError::Filesystem(error.to_string())),
        }
    }

    fn resolve_missing_leaf(&self, candidate: &Path) -> Result<PathBuf, PathError> {
        let parent = candidate.parent().ok_or(PathError::MissingAncestor)?;
        let parent = fs::canonicalize(parent).map_err(|error| {
            if error.kind() == std::io::ErrorKind::NotFound {
                PathError::MissingAncestor
            } else {
                PathError::Filesystem(error.to_string())
            }
        })?;
        self.ensure_inside(&parent)?;

        let file_name = candidate.file_name().ok_or(PathError::InvalidComponent)?;
        Ok(parent.join(file_name))
    }

    fn ensure_inside(&self, path: &Path) -> Result<PathBuf, PathError> {
        if path.starts_with(&self.root) {
            Ok(path.to_path_buf())
        } else {
            Err(PathError::OutsideWorkspace)
        }
    }
}

fn contains_percent_escape(path: &str) -> bool {
    path.as_bytes().windows(3).any(|bytes| {
        bytes[0] == b'%' && bytes[1].is_ascii_hexdigit() && bytes[2].is_ascii_hexdigit()
    })
}
