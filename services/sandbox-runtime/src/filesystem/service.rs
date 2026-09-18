use std::{path::Path, sync::Arc};

use serde::Serialize;
use thiserror::Error;
use tokio::io::AsyncReadExt;

use super::paths::{PathError, PathGuard};

pub const DEFAULT_MAX_FILE_BYTES: usize = 10 * 1024 * 1024;
const MAX_DIRECTORY_ENTRIES: usize = 10_000;

#[derive(Clone)]
pub struct FileService {
    guard: Arc<PathGuard>,
    max_file_bytes: usize,
}

#[derive(Debug, Error)]
pub enum FileError {
    #[error(transparent)]
    Path(#[from] PathError),
    #[error("file was not found")]
    NotFound,
    #[error("file body exceeds the configured limit of {limit} bytes")]
    BodyTooLarge { limit: usize },
    #[error("path is a directory")]
    IsDirectory,
    #[error("path is not a directory")]
    NotDirectory,
    #[error("directory is not empty")]
    DirectoryNotEmpty,
    #[error("directory contains too many entries")]
    TooManyEntries,
    #[error("file operation failed: {0}")]
    Filesystem(String),
}

impl FileError {
    pub fn code(&self) -> &'static str {
        match self {
            Self::Path(error) => error.code(),
            Self::NotFound => "file_not_found",
            Self::BodyTooLarge { .. } => "body_too_large",
            Self::IsDirectory
            | Self::NotDirectory
            | Self::DirectoryNotEmpty
            | Self::TooManyEntries
            | Self::Filesystem(_) => "runtime_filesystem_error",
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum FileKind {
    File,
    Directory,
    Symlink,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub struct FileEntry {
    pub path: String,
    pub kind: FileKind,
    #[serde(rename = "byteSize", skip_serializing_if = "Option::is_none")]
    pub byte_size: Option<u64>,
}

impl FileService {
    pub fn new(guard: PathGuard) -> Self {
        Self::with_max_file_bytes(guard, DEFAULT_MAX_FILE_BYTES)
    }

    pub fn with_max_file_bytes(guard: PathGuard, max_file_bytes: usize) -> Self {
        Self {
            guard: Arc::new(guard),
            max_file_bytes,
        }
    }

    pub fn max_file_bytes(&self) -> usize {
        self.max_file_bytes
    }

    pub async fn read(&self, user_path: &str) -> Result<Vec<u8>, FileError> {
        let path = self.guard.resolve_for_operation(user_path)?;
        let metadata = tokio::fs::metadata(&path).await.map_err(map_io)?;
        if metadata.is_dir() {
            return Err(FileError::IsDirectory);
        }
        if metadata.len() > self.max_file_bytes as u64 {
            return Err(FileError::BodyTooLarge {
                limit: self.max_file_bytes,
            });
        }

        let file = tokio::fs::File::open(path).await.map_err(map_io)?;
        let mut content = Vec::with_capacity(metadata.len() as usize);
        file.take(self.max_file_bytes as u64 + 1)
            .read_to_end(&mut content)
            .await
            .map_err(map_io)?;
        if content.len() > self.max_file_bytes {
            return Err(FileError::BodyTooLarge {
                limit: self.max_file_bytes,
            });
        }
        Ok(content)
    }

    pub async fn write(&self, user_path: &str, content: &[u8]) -> Result<FileEntry, FileError> {
        if content.len() > self.max_file_bytes {
            return Err(FileError::BodyTooLarge {
                limit: self.max_file_bytes,
            });
        }

        let path = self.guard.resolve_for_operation(user_path)?;
        if let Ok(metadata) = tokio::fs::symlink_metadata(&path).await {
            if metadata.is_dir() {
                return Err(FileError::IsDirectory);
            }
        }
        tokio::fs::write(&path, content).await.map_err(map_io)?;
        self.entry(&path).await
    }

    pub async fn list(&self, user_path: &str) -> Result<Vec<FileEntry>, FileError> {
        let path = self.guard.resolve_for_operation(user_path)?;
        let metadata = tokio::fs::metadata(&path).await.map_err(map_io)?;
        if !metadata.is_dir() {
            return Err(FileError::NotDirectory);
        }

        let mut entries = Vec::new();
        let mut directory = tokio::fs::read_dir(&path).await.map_err(map_io)?;
        while let Some(entry) = directory.next_entry().await.map_err(map_io)? {
            if entries.len() == MAX_DIRECTORY_ENTRIES {
                return Err(FileError::TooManyEntries);
            }
            entries.push(self.entry(&entry.path()).await?);
        }
        entries.sort_by(|left, right| left.path.cmp(&right.path));
        Ok(entries)
    }

    pub async fn delete(&self, user_path: &str) -> Result<(), FileError> {
        let path = self.guard.resolve_for_operation(user_path)?;
        let metadata = tokio::fs::symlink_metadata(&path).await.map_err(map_io)?;
        if metadata.is_dir() {
            match tokio::fs::remove_dir(&path).await {
                Ok(()) => Ok(()),
                Err(error) if error.kind() == std::io::ErrorKind::DirectoryNotEmpty => {
                    Err(FileError::DirectoryNotEmpty)
                }
                Err(error) => Err(map_io(error)),
            }
        } else {
            tokio::fs::remove_file(&path).await.map_err(map_io)
        }
    }

    async fn entry(&self, path: &Path) -> Result<FileEntry, FileError> {
        let metadata = tokio::fs::symlink_metadata(path).await.map_err(map_io)?;
        let file_type = metadata.file_type();
        let (kind, byte_size) = if file_type.is_symlink() {
            (FileKind::Symlink, None)
        } else if file_type.is_dir() {
            (FileKind::Directory, None)
        } else if file_type.is_file() {
            (FileKind::File, Some(metadata.len()))
        } else {
            return Err(FileError::Filesystem(
                "unsupported workspace entry type".to_owned(),
            ));
        };

        Ok(FileEntry {
            path: self.guard.virtual_path(path)?,
            kind,
            byte_size,
        })
    }
}

fn map_io(error: std::io::Error) -> FileError {
    if error.kind() == std::io::ErrorKind::NotFound {
        FileError::NotFound
    } else {
        FileError::Filesystem(error.to_string())
    }
}
