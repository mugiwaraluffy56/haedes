use std::{
    collections::HashMap,
    time::{Duration, Instant},
};

use haedes_sandbox_runtime::filesystem::PathGuard;
use haedes_sandbox_runtime::process::{CommandError, CommandRequest, CommandRunner};
use tempfile::tempdir;

fn request(command: &str) -> CommandRequest {
    CommandRequest {
        command: command.to_owned(),
        cwd: None,
        environment: HashMap::new(),
        timeout: Duration::from_secs(2),
    }
}

fn runner() -> (tempfile::TempDir, CommandRunner) {
    let workspace = tempdir().unwrap();
    let runner = CommandRunner::new(PathGuard::new(workspace.path()).unwrap());
    (workspace, runner)
}

#[tokio::test]
async fn commands_capture_success_failure_and_shell_output() {
    let (_workspace, runner) = runner();
    let result = runner
        .run(request("printf 'out'; printf 'err' >&2; exit 7"))
        .await
        .unwrap();

    assert_eq!(result.stdout, b"out");
    assert_eq!(result.stderr, b"err");
    assert_eq!(result.exit_code, Some(7));
    assert!(!result.timed_out);
}

#[tokio::test]
async fn commands_use_workspace_cwd_and_injected_environment() {
    let (workspace, runner) = runner();
    let nested = workspace.path().join("nested");
    std::fs::create_dir(&nested).unwrap();
    let mut command = request("printf '%s:%s' \"$HAEDES_TEST\" \"$(basename \"$PWD\")\"");
    command.cwd = Some("/workspace/nested".to_owned());
    command
        .environment
        .insert("HAEDES_TEST".to_owned(), "ok".to_owned());

    let result = runner.run(command).await.unwrap();
    assert_eq!(result.stdout, b"ok:nested");
}

#[tokio::test]
async fn commands_do_not_inherit_cloud_credentials() {
    let (_workspace, runner) = runner();
    let result = runner
        .run(request(
            "printf '%s:%s' \"${AWS_ACCESS_KEY_ID-unset}\" \"${AWS_PROFILE-unset}\"",
        ))
        .await
        .unwrap();

    assert_eq!(result.stdout, b"unset:unset");
}

#[tokio::test]
async fn command_timeout_terminates_the_process_group() {
    let (_workspace, runner) = runner();
    let mut command = request("sleep 10 & wait");
    command.timeout = Duration::from_millis(100);
    let started = Instant::now();

    let result = runner.run(command).await.unwrap();

    assert!(result.timed_out);
    assert!(result.exit_code.is_none());
    assert!(started.elapsed() < Duration::from_secs(2));
}

#[tokio::test]
async fn command_timeout_terminates_descendants() {
    let (workspace, runner) = runner();
    let mut command = request(
        "(sleep 1; touch descendant-marker) & child=$!; printf '%s' \"$child\" > descendant.pid; wait",
    );
    command.timeout = Duration::from_millis(100);

    let result = runner.run(command).await.unwrap();
    assert!(result.timed_out);

    tokio::time::sleep(Duration::from_millis(1200)).await;
    assert!(!workspace.path().join("descendant-marker").exists());
}

#[tokio::test]
async fn command_output_is_bounded() {
    let workspace = tempdir().unwrap();
    let runner = CommandRunner::with_limits(
        PathGuard::new(workspace.path()).unwrap(),
        1024,
        4,
        Duration::from_secs(2),
    );

    let error = runner.run(request("printf '12345'")).await.unwrap_err();
    assert!(matches!(error, CommandError::OutputTooLarge { limit: 4 }));
}

#[tokio::test]
async fn command_inputs_and_environment_limits_are_validated() {
    let (_workspace, runner) = runner();
    let mut empty = request("");
    empty.timeout = Duration::from_secs(1);
    assert!(matches!(
        runner.run(empty).await,
        Err(CommandError::EmptyCommand)
    ));

    let mut invalid = request("printf ok");
    invalid
        .environment
        .insert("BAD=KEY".to_owned(), "x".to_owned());
    assert!(matches!(
        runner.run(invalid).await,
        Err(CommandError::InvalidEnvironmentKey)
    ));
}
