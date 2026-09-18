use std::{
    collections::{HashMap, HashSet},
    fs::{self, File},
    io::{self, Cursor, Read, Write},
    path::{Component, Path, PathBuf},
};

#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;

use sha2::{Digest, Sha256};
use tar::{Builder, EntryType, Header};
use thiserror::Error;
use zstd::stream::{read::Decoder, write::Encoder};

use crate::filesystem::{PathError, PathGuard};

pub const MAX_ARCHIVE_BYTES: usize = 100 * 1024 * 1024;
const ARCHIVE_ROOT: &str = "workspace";
const MEDIA_TYPE: &str = "application/vnd.haedes.workspace+tar.zstd";

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ArchiveInfo {
    pub byte_size: usize,
    pub sha256: String,
    pub media_type: &'static str,
}

#[derive(Debug, Error)]
pub enum ArchiveError {
    #[error("workspace path error: {0}")]
    Path(#[from] PathError),
    #[error("archive exceeds the configured limit of {limit} bytes")]
    ArchiveTooLarge { limit: usize },
    #[error("archive entry is unsafe: {0}")]
    EntryUnsafe(String),
    #[error("archive contains conflicting entries: {0}")]
    ConflictingEntry(String),
    #[error("archive checksum mismatch: expected {expected}, got {actual}")]
    ChecksumMismatch { expected: String, actual: String },
    #[error("archive format error: {0}")]
    Format(String),
    #[error("archive I/O error: {0}")]
    Io(String),
}

#[derive(Clone)]
pub struct ArchiveService {
    guard: PathGuard,
}

impl ArchiveService {
    pub fn new(guard: PathGuard) -> Self {
        Self { guard }
    }

    pub fn export(&self) -> Result<(Vec<u8>, ArchiveInfo), ArchiveError> {
        let mut compressed = Vec::new();
        {
            let encoder = Encoder::new(&mut compressed, 3).map_err(format_io)?;
            let mut builder = Builder::new(encoder);
            append_directory(&mut builder, self.guard.root(), Path::new(ARCHIVE_ROOT))?;
            let encoder = builder.into_inner().map_err(format_io)?;
            encoder.finish().map_err(format_io)?;
        }
        if compressed.len() > MAX_ARCHIVE_BYTES {
            return Err(ArchiveError::ArchiveTooLarge {
                limit: MAX_ARCHIVE_BYTES,
            });
        }
        let info = archive_info(&compressed);
        Ok((compressed, info))
    }

    pub fn restore(
        &self,
        archive: &[u8],
        expected_sha256: Option<&str>,
    ) -> Result<ArchiveInfo, ArchiveError> {
        if archive.len() > MAX_ARCHIVE_BYTES {
            return Err(ArchiveError::ArchiveTooLarge {
                limit: MAX_ARCHIVE_BYTES,
            });
        }
        let info = archive_info(archive);
        if let Some(expected) = expected_sha256 {
            if expected != info.sha256 {
                return Err(ArchiveError::ChecksumMismatch {
                    expected: expected.to_owned(),
                    actual: info.sha256,
                });
            }
        }

        let decoder = Decoder::new(Cursor::new(archive)).map_err(format_io)?;
        let mut tar = tar::Archive::new(decoder);
        let entries = collect_entries(&mut tar)?;
        validate_entries(&entries)?;
        apply_entries(&self.guard, &entries)?;
        Ok(info)
    }
}

#[derive(Debug)]
enum ArchiveEntry {
    Directory {
        path: PathBuf,
        mode: u32,
    },
    File {
        path: PathBuf,
        mode: u32,
        body: Vec<u8>,
    },
    Symlink {
        path: PathBuf,
        target: PathBuf,
    },
}

fn append_directory<W: Write>(
    builder: &mut Builder<W>,
    directory: &Path,
    archive_path: &Path,
) -> Result<(), ArchiveError> {
    let metadata = fs::symlink_metadata(directory).map_err(format_io)?;
    append_dir_header(builder, archive_path, metadata.permissions().mode())?;
    let mut children = fs::read_dir(directory)
        .map_err(format_io)?
        .collect::<Result<Vec<_>, _>>()
        .map_err(format_io)?;
    children.sort_by_key(|entry| entry.file_name());
    for child in children {
        let child_path = child.path();
        let child_archive_path = archive_path.join(child.file_name());
        let metadata = fs::symlink_metadata(&child_path).map_err(format_io)?;
        let file_type = metadata.file_type();
        if file_type.is_dir() {
            append_directory(builder, &child_path, &child_archive_path)?;
        } else if file_type.is_file() {
            let mut file = File::open(&child_path).map_err(format_io)?;
            let mut body = Vec::new();
            file.read_to_end(&mut body).map_err(format_io)?;
            append_file(
                builder,
                &child_archive_path,
                metadata.permissions().mode(),
                &body,
            )?;
        } else if file_type.is_symlink() {
            let target = fs::read_link(&child_path).map_err(format_io)?;
            validate_symlink(&child_archive_path, &target)?;
            append_symlink(builder, &child_archive_path, &target)?;
        } else {
            return Err(ArchiveError::EntryUnsafe(
                child_archive_path.display().to_string(),
            ));
        }
    }
    Ok(())
}

fn append_dir_header<W: Write>(
    builder: &mut Builder<W>,
    path: &Path,
    mode: u32,
) -> Result<(), ArchiveError> {
    let mut header = Header::new_gnu();
    header.set_entry_type(EntryType::Directory);
    header.set_mode(mode);
    header.set_size(0);
    header.set_path(path).map_err(format_io)?;
    header.set_cksum();
    builder.append(&header, io::empty()).map_err(format_io)
}

fn append_file<W: Write>(
    builder: &mut Builder<W>,
    path: &Path,
    mode: u32,
    body: &[u8],
) -> Result<(), ArchiveError> {
    let mut header = Header::new_gnu();
    header.set_entry_type(EntryType::Regular);
    header.set_mode(mode);
    header.set_size(body.len() as u64);
    header.set_path(path).map_err(format_io)?;
    header.set_cksum();
    builder.append(&header, body).map_err(format_io)
}

fn append_symlink<W: Write>(
    builder: &mut Builder<W>,
    path: &Path,
    target: &Path,
) -> Result<(), ArchiveError> {
    let mut header = Header::new_gnu();
    header.set_entry_type(EntryType::Symlink);
    header.set_size(0);
    header.set_path(path).map_err(format_io)?;
    header.set_link_name(target).map_err(format_io)?;
    header.set_cksum();
    builder.append(&header, io::empty()).map_err(format_io)
}

fn collect_entries<R: Read>(
    archive: &mut tar::Archive<R>,
) -> Result<Vec<ArchiveEntry>, ArchiveError> {
    let mut entries = Vec::new();
    for entry in archive.entries().map_err(format_io)? {
        let mut entry = entry.map_err(format_io)?;
        let path = entry.path().map_err(format_io)?.to_path_buf();
        let header = entry.header().clone();
        let entry_type = header.entry_type();
        if entry_type == EntryType::Directory {
            entries.push(ArchiveEntry::Directory {
                path,
                mode: header.mode().unwrap_or(0o755),
            });
        } else if entry_type == EntryType::Regular {
            let size =
                usize::try_from(header.size().map_err(format_io)?).unwrap_or(MAX_ARCHIVE_BYTES + 1);
            if size > MAX_ARCHIVE_BYTES {
                return Err(ArchiveError::ArchiveTooLarge {
                    limit: MAX_ARCHIVE_BYTES,
                });
            }
            let mut body = Vec::new();
            entry.read_to_end(&mut body).map_err(format_io)?;
            entries.push(ArchiveEntry::File {
                path,
                mode: header.mode().unwrap_or(0o644),
                body,
            });
        } else if entry_type == EntryType::Symlink {
            let target = header
                .link_name()
                .map_err(format_io)?
                .ok_or_else(|| ArchiveError::EntryUnsafe("symlink without a target".to_owned()))?
                .to_path_buf();
            entries.push(ArchiveEntry::Symlink { path, target });
        } else {
            return Err(ArchiveError::EntryUnsafe(path.display().to_string()));
        }
    }
    Ok(entries)
}

fn validate_entries(entries: &[ArchiveEntry]) -> Result<(), ArchiveError> {
    let mut paths = HashSet::new();
    let mut kinds = HashMap::new();
    for entry in entries {
        let path = entry.path();
        validate_archive_path(path)?;
        let key = path.to_string_lossy().to_string();
        if !paths.insert(key.clone()) {
            return Err(ArchiveError::ConflictingEntry(key));
        }
        kinds.insert(key, entry.kind());
        if let ArchiveEntry::Symlink { path, target } = entry {
            validate_symlink(path, target)?;
        }
    }
    for entry in entries {
        let mut parent = entry.path().parent();
        while let Some(path) = parent {
            if let Some(kind) = kinds.get(&path.to_string_lossy().to_string()) {
                if *kind != EntryKind::Directory {
                    return Err(ArchiveError::ConflictingEntry(path.display().to_string()));
                }
            }
            parent = path.parent();
        }
    }
    Ok(())
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum EntryKind {
    Directory,
    File,
    Symlink,
}

impl ArchiveEntry {
    fn path(&self) -> &Path {
        match self {
            Self::Directory { path, .. } | Self::File { path, .. } | Self::Symlink { path, .. } => {
                path
            }
        }
    }

    fn kind(&self) -> EntryKind {
        match self {
            Self::Directory { .. } => EntryKind::Directory,
            Self::File { .. } => EntryKind::File,
            Self::Symlink { .. } => EntryKind::Symlink,
        }
    }
}

fn validate_archive_path(path: &Path) -> Result<(), ArchiveError> {
    let mut components = path.components();
    if components.next() != Some(Component::Normal(std::ffi::OsStr::new(ARCHIVE_ROOT))) {
        return Err(ArchiveError::EntryUnsafe(path.display().to_string()));
    }
    for component in components {
        if !matches!(component, Component::Normal(_)) {
            return Err(ArchiveError::EntryUnsafe(path.display().to_string()));
        }
    }
    Ok(())
}

fn validate_symlink(path: &Path, target: &Path) -> Result<(), ArchiveError> {
    if target.is_absolute()
        || target
            .components()
            .any(|component| !matches!(component, Component::Normal(_) | Component::CurDir))
    {
        return Err(ArchiveError::EntryUnsafe(format!(
            "{} -> {}",
            path.display(),
            target.display()
        )));
    }
    Ok(())
}

fn apply_entries(guard: &PathGuard, entries: &[ArchiveEntry]) -> Result<(), ArchiveError> {
    let mut directories = entries
        .iter()
        .filter(|entry| matches!(entry, ArchiveEntry::Directory { .. }))
        .collect::<Vec<_>>();
    directories.sort_by_key(|entry| entry.path().components().count());
    for entry in directories {
        if let ArchiveEntry::Directory { path, mode } = entry {
            let destination = destination(guard, path)?;
            if let Ok(metadata) = fs::symlink_metadata(&destination) {
                if !metadata.is_dir() || metadata.file_type().is_symlink() {
                    return Err(ArchiveError::ConflictingEntry(path.display().to_string()));
                }
            } else {
                fs::create_dir(&destination).map_err(format_io)?;
            }
            fs::set_permissions(&destination, fs::Permissions::from_mode(*mode))
                .map_err(format_io)?;
        }
    }
    for entry in entries {
        match entry {
            ArchiveEntry::File { path, mode, body } => {
                let destination = destination(guard, path)?;
                if let Ok(metadata) = fs::symlink_metadata(&destination) {
                    if metadata.file_type().is_symlink() || metadata.is_dir() {
                        return Err(ArchiveError::ConflictingEntry(path.display().to_string()));
                    }
                }
                let mut file = File::create(&destination).map_err(format_io)?;
                file.write_all(body).map_err(format_io)?;
                fs::set_permissions(&destination, fs::Permissions::from_mode(*mode))
                    .map_err(format_io)?;
            }
            ArchiveEntry::Symlink { path, target } => {
                let destination = destination(guard, path)?;
                if fs::symlink_metadata(&destination).is_ok() {
                    return Err(ArchiveError::ConflictingEntry(path.display().to_string()));
                }
                #[cfg(unix)]
                std::os::unix::fs::symlink(target, destination).map_err(format_io)?;
                #[cfg(not(unix))]
                return Err(ArchiveError::EntryUnsafe(
                    "symlinks are unsupported on this platform".to_owned(),
                ));
            }
            ArchiveEntry::Directory { .. } => {}
        }
    }
    Ok(())
}

fn destination(guard: &PathGuard, archive_path: &Path) -> Result<PathBuf, ArchiveError> {
    let relative = archive_path
        .strip_prefix(ARCHIVE_ROOT)
        .map_err(|_| ArchiveError::EntryUnsafe(archive_path.display().to_string()))?;
    let virtual_path = Path::new("/workspace").join(relative);
    guard
        .resolve_for_operation(
            virtual_path
                .to_str()
                .ok_or_else(|| ArchiveError::EntryUnsafe(archive_path.display().to_string()))?,
        )
        .map_err(ArchiveError::Path)
}

fn archive_info(archive: &[u8]) -> ArchiveInfo {
    let digest = Sha256::digest(archive);
    ArchiveInfo {
        byte_size: archive.len(),
        sha256: hex::encode(digest),
        media_type: MEDIA_TYPE,
    }
}

fn format_io(error: impl ToString) -> ArchiveError {
    ArchiveError::Io(error.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::tempdir;

    #[test]
    fn round_trip_preserves_regular_files_and_directories() {
        let source = tempdir().unwrap();
        fs::create_dir(source.path().join("nested")).unwrap();
        fs::write(source.path().join("hello.txt"), b"hello").unwrap();
        fs::write(source.path().join("nested/data.txt"), b"payload").unwrap();

        let source_guard = PathGuard::new(source.path()).unwrap();
        let source_service = ArchiveService::new(source_guard);
        let (archive, expected_info) = source_service.export().unwrap();

        let destination = tempdir().unwrap();
        let destination_guard = PathGuard::new(destination.path()).unwrap();
        let destination_service = ArchiveService::new(destination_guard);
        let actual_info = destination_service
            .restore(&archive, Some(&expected_info.sha256))
            .unwrap();

        assert_eq!(actual_info, expected_info);
        assert_eq!(
            fs::read(destination.path().join("hello.txt")).unwrap(),
            b"hello"
        );
        assert_eq!(
            fs::read(destination.path().join("nested/data.txt")).unwrap(),
            b"payload"
        );
    }

    #[test]
    fn restore_rejects_checksum_mismatch_without_writing() {
        let source = tempdir().unwrap();
        fs::write(source.path().join("hello.txt"), b"hello").unwrap();
        let service = ArchiveService::new(PathGuard::new(source.path()).unwrap());
        let (archive, _) = service.export().unwrap();

        let destination = tempdir().unwrap();
        let destination_service = ArchiveService::new(PathGuard::new(destination.path()).unwrap());
        let error = destination_service.restore(&archive, Some("not-the-checksum"));

        assert!(matches!(error, Err(ArchiveError::ChecksumMismatch { .. })));
        assert!(!destination.path().join("hello.txt").exists());
    }

    #[test]
    fn restore_rejects_unsafe_paths_and_symlink_targets() {
        let workspace = tempdir().unwrap();
        let service = ArchiveService::new(PathGuard::new(workspace.path()).unwrap());

        for (path, target) in [
            (Path::new("/etc/passwd"), None),
            (Path::new("workspace/../escape"), None),
            (Path::new("workspace/link"), Some(Path::new("../outside"))),
        ] {
            let archive = malicious_archive(path, target);
            assert!(matches!(
                service.restore(&archive, None),
                Err(ArchiveError::EntryUnsafe(_))
            ));
        }
        assert!(fs::read_dir(workspace.path()).unwrap().next().is_none());
    }

    #[test]
    fn restore_rejects_conflicting_entries_before_writing() {
        let mut compressed = Vec::new();
        {
            let encoder = Encoder::new(&mut compressed, 3).unwrap();
            let mut builder = Builder::new(encoder);
            append_test_file(&mut builder, Path::new("workspace/conflict"), b"file");
            append_test_file(
                &mut builder,
                Path::new("workspace/conflict/child"),
                b"child",
            );
            let encoder = builder.into_inner().unwrap();
            encoder.finish().unwrap();
        }

        let workspace = tempdir().unwrap();
        let service = ArchiveService::new(PathGuard::new(workspace.path()).unwrap());
        assert!(matches!(
            service.restore(&compressed, None),
            Err(ArchiveError::ConflictingEntry(_))
        ));
        assert!(fs::read_dir(workspace.path()).unwrap().next().is_none());
    }

    fn malicious_archive(path: &Path, target: Option<&Path>) -> Vec<u8> {
        let mut compressed = Vec::new();
        {
            let encoder = Encoder::new(&mut compressed, 3).unwrap();
            let mut builder = Builder::new(encoder);
            let mut header = Header::new_gnu();
            header.set_entry_type(if target.is_some() {
                EntryType::Symlink
            } else {
                EntryType::Regular
            });
            header.set_mode(0o644);
            header.set_size(0);
            set_test_path(&mut header, path);
            if let Some(target) = target {
                header.set_link_name(target).unwrap();
            }
            header.set_cksum();
            builder.append(&header, io::empty()).unwrap();
            let encoder = builder.into_inner().unwrap();
            encoder.finish().unwrap();
        }
        compressed
    }

    fn append_test_file<W: Write>(builder: &mut Builder<W>, path: &Path, body: &[u8]) {
        let mut header = Header::new_gnu();
        header.set_entry_type(EntryType::Regular);
        header.set_mode(0o644);
        header.set_size(body.len() as u64);
        header.set_path(path).unwrap();
        header.set_cksum();
        builder.append(&header, body).unwrap();
    }

    fn set_test_path(header: &mut Header, path: &Path) {
        let path = path.to_string_lossy();
        let bytes = path.as_bytes();
        assert!(bytes.len() <= 100);
        let name = &mut header.as_mut_bytes()[..100];
        name.fill(0);
        name[..bytes.len()].copy_from_slice(bytes);
    }
}
