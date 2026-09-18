use std::{
    collections::HashMap,
    process::Stdio,
    sync::{
        atomic::{AtomicU64, Ordering},
        Arc, Mutex,
    },
    time::Duration,
};

use nix::{
    sys::signal::{killpg, Signal},
    unistd::Pid,
};
use thiserror::Error;
use tokio::{
    io::{AsyncRead, AsyncReadExt},
    process::Command,
    time::sleep,
};

use crate::{
    filesystem::{PathError, PathGuard},
    streaming::{CommandEventKind, CommandResultSummary, EventBus, EventError, EventReceiver},
};

pub const DEFAULT_MAX_COMMAND_BYTES: usize = 16 * 1024;
pub const DEFAULT_MAX_OUTPUT_BYTES: usize = 1024 * 1024;
pub const DEFAULT_MAX_TIMEOUT: Duration = Duration::from_secs(900);
const MAX_ENVIRONMENT_ENTRIES: usize = 32;
const MAX_ENVIRONMENT_VALUE_BYTES: usize = 4096;
const MAX_ENVIRONMENT_KEY_BYTES: usize = 256;
const SAFE_PATH: &str = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin";

#[derive(Clone)]
pub struct CommandRunner {
    guard: Arc<PathGuard>,
    max_command_bytes: usize,
    max_output_bytes: usize,
    max_timeout: Duration,
    next_id: Arc<AtomicU64>,
    commands: Arc<Mutex<HashMap<String, EventBus>>>,
}

#[derive(Clone)]
pub struct CommandHandle {
    id: String,
    events: EventBus,
}

impl CommandHandle {
    pub fn id(&self) -> &str {
        &self.id
    }

    pub fn events(&self, after_sequence: u64) -> Result<EventReceiver, EventError> {
        self.events.subscribe(after_sequence)
    }
}

#[derive(Debug, Clone)]
pub struct CommandRequest {
    pub command: String,
    pub cwd: Option<String>,
    pub environment: HashMap<String, String>,
    pub timeout: Duration,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CommandResult {
    pub exit_code: Option<i32>,
    pub signal: Option<String>,
    pub timed_out: bool,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
}

#[derive(Debug, Error)]
pub enum CommandError {
    #[error(transparent)]
    Events(#[from] EventError),
    #[error(transparent)]
    Path(#[from] PathError),
    #[error("command must not be empty")]
    EmptyCommand,
    #[error("command exceeds the configured limit of {limit} bytes")]
    CommandTooLarge { limit: usize },
    #[error("command timeout must be greater than zero")]
    InvalidTimeout,
    #[error("command timeout exceeds the configured limit of {limit:?}")]
    TimeoutTooLarge { limit: Duration },
    #[error("command environment contains too many entries")]
    EnvironmentTooLarge,
    #[error("command environment key is invalid")]
    InvalidEnvironmentKey,
    #[error("command environment value exceeds the configured limit of {limit} bytes")]
    EnvironmentValueTooLarge { limit: usize },
    #[error("command output exceeds the configured limit of {limit} bytes")]
    OutputTooLarge { limit: usize },
    #[error("working directory is not a directory")]
    InvalidWorkingDirectory,
    #[error("failed to start command: {0}")]
    Spawn(String),
    #[error("command process failed: {0}")]
    Process(String),
}

impl CommandError {
    pub fn code(&self) -> &'static str {
        match self {
            Self::Path(error) => error.code(),
            Self::Events(_) => "runtime_unavailable",
            Self::OutputTooLarge { .. } => "command_output_limit",
            Self::InvalidTimeout | Self::TimeoutTooLarge { .. } => "command_timeout",
            Self::EmptyCommand
            | Self::CommandTooLarge { .. }
            | Self::EnvironmentTooLarge
            | Self::InvalidEnvironmentKey
            | Self::EnvironmentValueTooLarge { .. }
            | Self::InvalidWorkingDirectory
            | Self::Spawn(_)
            | Self::Process(_) => "runtime_unavailable",
        }
    }
}

impl CommandRunner {
    pub fn new(guard: PathGuard) -> Self {
        Self {
            guard: Arc::new(guard),
            max_command_bytes: DEFAULT_MAX_COMMAND_BYTES,
            max_output_bytes: DEFAULT_MAX_OUTPUT_BYTES,
            max_timeout: DEFAULT_MAX_TIMEOUT,
            next_id: Arc::new(AtomicU64::new(1)),
            commands: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn with_limits(
        guard: PathGuard,
        max_command_bytes: usize,
        max_output_bytes: usize,
        max_timeout: Duration,
    ) -> Self {
        Self {
            guard: Arc::new(guard),
            max_command_bytes,
            max_output_bytes,
            max_timeout,
            next_id: Arc::new(AtomicU64::new(1)),
            commands: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub async fn run(&self, request: CommandRequest) -> Result<CommandResult, CommandError> {
        self.execute(request, None).await
    }

    pub fn start(&self, request: CommandRequest) -> Result<CommandHandle, CommandError> {
        self.validate_request(&request)?;
        let id = format!("cmd_{}", self.next_id.fetch_add(1, Ordering::Relaxed));
        let events = EventBus::new(id.clone());
        self.commands
            .lock()
            .expect("command registry lock poisoned")
            .insert(id.clone(), events.clone());

        let runner = self.clone();
        let event_bus = events.clone();
        tokio::spawn(async move {
            let _ = event_bus.publish(CommandEventKind::Started);
            match runner.execute(request, Some(event_bus.clone())).await {
                Ok(result) => {
                    let _ = event_bus.publish(CommandEventKind::Completed(CommandResultSummary {
                        exit_code: result.exit_code,
                        signal: result.signal,
                        timed_out: result.timed_out,
                    }));
                }
                Err(error) => {
                    let _ = event_bus.publish(CommandEventKind::Failed {
                        code: error.code().to_owned(),
                        message: error.to_string(),
                    });
                }
            }
        });

        Ok(CommandHandle { id, events })
    }

    pub fn events(
        &self,
        command_id: &str,
        after_sequence: u64,
    ) -> Result<EventReceiver, CommandError> {
        let events = self
            .commands
            .lock()
            .expect("command registry lock poisoned")
            .get(command_id)
            .cloned()
            .ok_or_else(|| CommandError::Process("command was not found".to_owned()))?;
        Ok(events.subscribe(after_sequence)?)
    }

    async fn execute(
        &self,
        request: CommandRequest,
        events: Option<EventBus>,
    ) -> Result<CommandResult, CommandError> {
        self.validate_request(&request)?;
        let cwd = self.resolve_cwd(request.cwd.as_deref()).await?;

        let mut command = Command::new("sh");
        command
            .arg("-lc")
            .arg(&request.command)
            .current_dir(cwd)
            .env_clear()
            .env("PATH", SAFE_PATH)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .kill_on_drop(true);
        for (key, value) in &request.environment {
            command.env(key, value);
        }
        #[cfg(unix)]
        command.process_group(0);

        let mut child = command
            .spawn()
            .map_err(|error| CommandError::Spawn(error.to_string()))?;
        let process_id = child
            .id()
            .ok_or_else(|| CommandError::Spawn("command did not expose a process id".to_owned()))?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| CommandError::Spawn("command stdout was not piped".to_owned()))?;
        let stderr = child
            .stderr
            .take()
            .ok_or_else(|| CommandError::Spawn("command stderr was not piped".to_owned()))?;

        let mut stdout_task = tokio::spawn(read_stream(
            stdout,
            self.max_output_bytes,
            events.clone(),
            true,
        ));
        let mut stderr_task =
            tokio::spawn(read_stream(stderr, self.max_output_bytes, events, false));
        let mut wait_task = tokio::spawn(async move { child.wait().await });
        let mut timeout = Box::pin(sleep(request.timeout));
        let mut stdout_done = false;
        let mut stderr_done = false;
        let mut wait_done = false;
        let mut stdout_result = None;
        let mut stderr_result = None;
        let mut wait_result = None;
        let mut timed_out = false;
        let mut terminated = false;

        while !(stdout_done && stderr_done && wait_done) {
            tokio::select! {
                result = &mut stdout_task, if !stdout_done => {
                    stdout_done = true;
                    stdout_result = Some(join_stream(result, self.max_output_bytes));
                    if stream_failed(&stdout_result) && !terminated {
                        terminated = true;
                        kill_process_group(process_id);
                    }
                }
                result = &mut stderr_task, if !stderr_done => {
                    stderr_done = true;
                    stderr_result = Some(join_stream(result, self.max_output_bytes));
                    if stream_failed(&stderr_result) && !terminated {
                        terminated = true;
                        kill_process_group(process_id);
                    }
                }
                result = &mut wait_task, if !wait_done => {
                    wait_done = true;
                    wait_result = Some(result.map_err(|error| CommandError::Process(error.to_string()))?);
                }
                _ = &mut timeout, if !terminated => {
                    terminated = true;
                    timed_out = true;
                    kill_process_group(process_id);
                }
            }
        }

        let stdout = stdout_result
            .ok_or_else(|| CommandError::Process("stdout task did not finish".to_owned()))??;
        let stderr = stderr_result
            .ok_or_else(|| CommandError::Process("stderr task did not finish".to_owned()))??;
        let status = wait_result
            .ok_or_else(|| CommandError::Process("process wait did not finish".to_owned()))?
            .map_err(|error| CommandError::Process(error.to_string()))?;

        Ok(CommandResult {
            exit_code: if timed_out { None } else { status.code() },
            signal: process_signal(&status),
            timed_out,
            stdout,
            stderr,
        })
    }

    async fn resolve_cwd(&self, cwd: Option<&str>) -> Result<std::path::PathBuf, CommandError> {
        let path = match cwd {
            Some(cwd) => self.guard.resolve_for_operation(cwd)?,
            None => self.guard.root().to_path_buf(),
        };
        let metadata = tokio::fs::metadata(&path)
            .await
            .map_err(|error| CommandError::Process(error.to_string()))?;
        if !metadata.is_dir() {
            return Err(CommandError::InvalidWorkingDirectory);
        }
        Ok(path)
    }

    fn validate_request(&self, request: &CommandRequest) -> Result<(), CommandError> {
        if request.command.is_empty() {
            return Err(CommandError::EmptyCommand);
        }
        if request.command.len() > self.max_command_bytes {
            return Err(CommandError::CommandTooLarge {
                limit: self.max_command_bytes,
            });
        }
        if request.timeout.is_zero() {
            return Err(CommandError::InvalidTimeout);
        }
        if request.timeout > self.max_timeout {
            return Err(CommandError::TimeoutTooLarge {
                limit: self.max_timeout,
            });
        }
        if request.environment.len() > MAX_ENVIRONMENT_ENTRIES {
            return Err(CommandError::EnvironmentTooLarge);
        }
        for (key, value) in &request.environment {
            if key.is_empty()
                || key.len() > MAX_ENVIRONMENT_KEY_BYTES
                || key.contains('=')
                || key.as_bytes().contains(&0)
            {
                return Err(CommandError::InvalidEnvironmentKey);
            }
            if value.len() > MAX_ENVIRONMENT_VALUE_BYTES || value.as_bytes().contains(&0) {
                return Err(CommandError::EnvironmentValueTooLarge {
                    limit: MAX_ENVIRONMENT_VALUE_BYTES,
                });
            }
        }
        if self.max_output_bytes == 0 {
            return Err(CommandError::OutputTooLarge { limit: 0 });
        }
        Ok(())
    }
}

#[derive(Debug)]
enum StreamError {
    OutputLimit,
    Io(String),
    Event(String),
}

async fn read_stream<R>(
    mut stream: R,
    limit: usize,
    events: Option<EventBus>,
    stdout: bool,
) -> Result<Vec<u8>, StreamError>
where
    R: AsyncRead + Unpin,
{
    let mut output = Vec::new();
    let mut buffer = vec![0; 8192];
    loop {
        let size = stream
            .read(&mut buffer)
            .await
            .map_err(|error| StreamError::Io(error.to_string()))?;
        if size == 0 {
            break;
        }
        if output.len().saturating_add(size) > limit {
            return Err(StreamError::OutputLimit);
        }
        output.extend_from_slice(&buffer[..size]);
        if let Some(events) = &events {
            let kind = if stdout {
                CommandEventKind::Stdout(buffer[..size].to_vec())
            } else {
                CommandEventKind::Stderr(buffer[..size].to_vec())
            };
            events
                .publish(kind)
                .map_err(|error| StreamError::Event(error.to_string()))?;
        }
    }
    Ok(output)
}

fn join_stream(
    result: Result<Result<Vec<u8>, StreamError>, tokio::task::JoinError>,
    limit: usize,
) -> Result<Vec<u8>, CommandError> {
    match result {
        Ok(Ok(output)) => Ok(output),
        Ok(Err(StreamError::OutputLimit)) => Err(CommandError::OutputTooLarge { limit }),
        Ok(Err(StreamError::Io(error))) => Err(CommandError::Process(error)),
        Ok(Err(StreamError::Event(error))) => Err(CommandError::Process(error)),
        Err(error) => Err(CommandError::Process(error.to_string())),
    }
}

fn stream_failed(result: &Option<Result<Vec<u8>, CommandError>>) -> bool {
    result.as_ref().is_some_and(|value| value.is_err())
}

fn kill_process_group(process_id: u32) {
    #[cfg(unix)]
    {
        let _ = killpg(Pid::from_raw(process_id as i32), Signal::SIGKILL);
    }
}

#[cfg(unix)]
fn process_signal(status: &std::process::ExitStatus) -> Option<String> {
    use std::os::unix::process::ExitStatusExt;

    status.signal().map(|signal| format!("SIG{signal}"))
}

#[cfg(not(unix))]
fn process_signal(_: &std::process::ExitStatus) -> Option<String> {
    None
}
