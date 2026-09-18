use std::{collections::HashMap, time::Duration};

use haedes_sandbox_runtime::{
    filesystem::PathGuard,
    process::{CommandRequest, CommandRunner},
    streaming::{CommandEvent, CommandEventKind, EventBus, EventError},
};
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

async fn collect(
    mut receiver: haedes_sandbox_runtime::streaming::EventReceiver,
) -> Vec<CommandEvent> {
    let mut events = Vec::new();
    while let Some(event) = receiver.recv().await.unwrap() {
        events.push(event);
    }
    events
}

#[tokio::test]
async fn command_events_are_monotonic_and_replay_after_reconnect() {
    let (_workspace, runner) = runner();
    let handle = runner
        .start(request("printf 'out'; printf 'err' >&2"))
        .unwrap();
    let events = collect(handle.events(0).unwrap()).await;

    assert_eq!(events.len(), 4);
    assert!(matches!(&events[0].kind, CommandEventKind::Started));
    assert!(events[1..3]
        .iter()
        .any(|event| matches!(&event.kind, CommandEventKind::Stdout(data) if data == b"out")));
    assert!(events[1..3]
        .iter()
        .any(|event| matches!(&event.kind, CommandEventKind::Stderr(data) if data == b"err")));
    assert!(matches!(&events[3].kind, CommandEventKind::Completed(_)));
    assert_eq!(
        events
            .iter()
            .map(|event| event.sequence)
            .collect::<Vec<_>>(),
        vec![1, 2, 3, 4]
    );

    let replayed = collect(handle.events(events[1].sequence).unwrap()).await;
    assert_eq!(replayed, events[2..]);
}

#[tokio::test]
async fn output_limits_and_runtime_failures_are_terminal_failed_events() {
    let workspace = tempdir().unwrap();
    let limited_runner = CommandRunner::with_limits(
        PathGuard::new(workspace.path()).unwrap(),
        1024,
        4,
        Duration::from_secs(2),
    );
    let handle = limited_runner.start(request("printf '12345'")).unwrap();
    let events = collect(handle.events(0).unwrap()).await;
    assert!(matches!(
        events.last().map(|event| &event.kind),
        Some(CommandEventKind::Failed { code, .. }) if code == "command_output_limit"
    ));

    let (_workspace, command_runner) = runner();
    let mut failed_request = request("printf ok");
    failed_request.cwd = Some("/workspace/missing".to_owned());
    let handle = command_runner.start(failed_request).unwrap();
    let events = collect(handle.events(0).unwrap()).await;
    assert!(matches!(
        events.last().map(|event| &event.kind),
        Some(CommandEventKind::Failed { code, .. }) if code == "runtime_unavailable"
    ));
}

#[tokio::test]
async fn event_bus_reports_replay_loss_and_rejects_events_after_terminal() {
    let bus = EventBus::with_capacity("cmd_test", 2);
    bus.publish(CommandEventKind::Started).unwrap();
    bus.publish(CommandEventKind::Stdout(b"one".to_vec()))
        .unwrap();
    bus.publish(CommandEventKind::Stdout(b"two".to_vec()))
        .unwrap();
    assert!(matches!(
        bus.subscribe(0),
        Err(EventError::ReplayUnavailable(0))
    ));

    bus.publish(CommandEventKind::Completed(
        haedes_sandbox_runtime::streaming::CommandResultSummary {
            exit_code: Some(0),
            signal: None,
            timed_out: false,
        },
    ))
    .unwrap();
    assert!(matches!(
        bus.publish(CommandEventKind::Started),
        Err(EventError::PublishAfterTerminal)
    ));
}
