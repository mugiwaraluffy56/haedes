use std::fs;

use haedes_sandbox_runtime::filesystem::{FileError, FileKind, FileService, PathGuard};
use tempfile::tempdir;

#[tokio::test]
async fn file_service_reads_writes_lists_and_deletes_entries() {
    let workspace = tempdir().unwrap();
    fs::create_dir(workspace.path().join("src")).unwrap();
    fs::write(workspace.path().join("src/main.rs"), "fn main() {}").unwrap();
    let service = FileService::new(PathGuard::new(workspace.path()).unwrap());

    assert_eq!(
        service.read("/workspace/src/main.rs").await.unwrap(),
        b"fn main() {}"
    );

    let entry = service
        .write("notes.txt", "hédes".as_bytes())
        .await
        .unwrap();
    assert_eq!(entry.path, "/workspace/notes.txt");
    assert_eq!(entry.kind, FileKind::File);
    assert_eq!(entry.byte_size, Some("hédes".len() as u64));

    let entries = service.list("/workspace").await.unwrap();
    assert_eq!(
        entries
            .iter()
            .map(|entry| entry.path.as_str())
            .collect::<Vec<_>>(),
        vec!["/workspace/notes.txt", "/workspace/src",]
    );
    assert!(matches!(
        service.delete("src").await,
        Err(FileError::DirectoryNotEmpty)
    ));

    service.delete("/workspace/notes.txt").await.unwrap();
    assert!(matches!(
        service.read("notes.txt").await,
        Err(FileError::NotFound)
    ));
}

#[tokio::test]
async fn file_service_handles_empty_files_and_missing_entries() {
    let workspace = tempdir().unwrap();
    let service = FileService::new(PathGuard::new(workspace.path()).unwrap());

    service.write("empty.txt", &[]).await.unwrap();
    assert!(service.read("empty.txt").await.unwrap().is_empty());
    assert!(matches!(
        service.read("missing.txt").await,
        Err(FileError::NotFound)
    ));
    assert!(matches!(
        service.delete("missing.txt").await,
        Err(FileError::NotFound)
    ));
}

#[tokio::test]
async fn file_service_enforces_the_body_limit() {
    let workspace = tempdir().unwrap();
    let service = FileService::with_max_file_bytes(PathGuard::new(workspace.path()).unwrap(), 4);

    assert!(matches!(
        service.write("large.txt", b"12345").await,
        Err(FileError::BodyTooLarge { limit: 4 })
    ));
    fs::write(workspace.path().join("large.txt"), b"12345").unwrap();
    assert!(matches!(
        service.read("large.txt").await,
        Err(FileError::BodyTooLarge { limit: 4 })
    ));
}

#[cfg(unix)]
#[tokio::test]
async fn file_service_reports_symlinks_and_does_not_follow_them_on_delete() {
    use std::os::unix::fs::symlink;

    let workspace = tempdir().unwrap();
    let outside = tempdir().unwrap();
    fs::write(workspace.path().join("inside.txt"), b"safe").unwrap();
    fs::write(outside.path().join("secret.txt"), b"secret").unwrap();
    symlink(
        outside.path().join("secret.txt"),
        workspace.path().join("outside-link"),
    )
    .unwrap();
    symlink(
        workspace.path().join("inside.txt"),
        workspace.path().join("inside-link"),
    )
    .unwrap();
    let service = FileService::new(PathGuard::new(workspace.path()).unwrap());

    let entries = service.list("/workspace").await.unwrap();
    let outside_entry = entries
        .iter()
        .find(|entry| entry.path == "/workspace/outside-link")
        .unwrap();
    assert_eq!(outside_entry.kind, FileKind::Symlink);
    assert!(matches!(
        service.read("outside-link").await,
        Err(FileError::Path(_))
    ));

    service.delete("inside-link").await.unwrap();
    assert!(workspace.path().join("inside.txt").exists());
    assert!(!workspace.path().join("inside-link").exists());
    assert!(outside.path().join("secret.txt").exists());
}
