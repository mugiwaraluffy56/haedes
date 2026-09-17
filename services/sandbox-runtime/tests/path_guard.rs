use std::fs;

use haedes_sandbox_runtime::filesystem::{PathError, PathGuard};
use tempfile::tempdir;

#[test]
fn relative_and_absolute_workspace_paths_resolve_inside_root() {
    let workspace = tempdir().unwrap();
    fs::create_dir(workspace.path().join("src")).unwrap();
    fs::write(workspace.path().join("src/main.rs"), "fn main() {}").unwrap();
    let guard = PathGuard::new(workspace.path()).unwrap();

    assert_eq!(
        guard.resolve("src/main.rs").unwrap(),
        workspace.path().join("src/main.rs")
    );
    assert_eq!(
        guard.resolve("/workspace/src/main.rs").unwrap(),
        workspace.path().join("src/main.rs")
    );
    assert_eq!(
        guard
            .resolve(workspace.path().join("src/main.rs").to_str().unwrap())
            .unwrap(),
        workspace.path().join("src/main.rs")
    );
}

#[test]
fn traversal_encoded_paths_nul_and_outside_paths_are_rejected() {
    let workspace = tempdir().unwrap();
    let outside = tempdir().unwrap();
    let guard = PathGuard::new(workspace.path()).unwrap();

    for path in [
        "../outside.txt",
        "/workspace/../outside.txt",
        "/workspace/%2e%2e/outside.txt",
        "/workspace/inside%2F..%2Foutside.txt",
    ] {
        assert!(matches!(
            guard.resolve(path),
            Err(PathError::Traversal | PathError::EncodedPath)
        ));
    }
    assert!(matches!(
        guard.resolve("/workspace/inside\0.txt"),
        Err(PathError::NulByte)
    ));
    assert!(matches!(
        guard.resolve(outside.path().to_str().unwrap()),
        Err(PathError::OutsideWorkspace)
    ));
}

#[test]
fn missing_ancestors_are_rejected_but_a_missing_leaf_is_safe() {
    let workspace = tempdir().unwrap();
    let guard = PathGuard::new(workspace.path()).unwrap();

    let missing_leaf = guard.resolve("new.txt").unwrap();
    assert_eq!(missing_leaf, workspace.path().join("new.txt"));
    assert!(matches!(
        guard.resolve("missing/new.txt"),
        Err(PathError::MissingAncestor)
    ));
}

#[cfg(unix)]
#[test]
fn symlinks_must_resolve_inside_the_workspace() {
    use std::os::unix::fs::symlink;

    let workspace = tempdir().unwrap();
    let outside = tempdir().unwrap();
    fs::write(workspace.path().join("inside.txt"), "safe").unwrap();
    fs::write(outside.path().join("secret.txt"), "secret").unwrap();
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
    let guard = PathGuard::new(workspace.path()).unwrap();

    assert!(matches!(
        guard.resolve("outside-link"),
        Err(PathError::UnsafeSymlink)
    ));
    assert_eq!(
        guard.resolve("inside-link").unwrap(),
        workspace.path().join("inside.txt")
    );
}
