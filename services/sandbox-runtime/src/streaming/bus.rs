use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};

use thiserror::Error;
use tokio::sync::broadcast;

const DEFAULT_HISTORY_CAPACITY: usize = 1024;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CommandResultSummary {
    pub exit_code: Option<i32>,
    pub signal: Option<String>,
    pub timed_out: bool,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum CommandEventKind {
    Started,
    Stdout(Vec<u8>),
    Stderr(Vec<u8>),
    Completed(CommandResultSummary),
    Failed { code: String, message: String },
}

impl CommandEventKind {
    fn is_terminal(&self) -> bool {
        matches!(self, Self::Completed(_) | Self::Failed { .. })
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CommandEvent {
    pub command_id: String,
    pub sequence: u64,
    pub kind: CommandEventKind,
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum EventError {
    #[error("event history is no longer available from sequence {0}")]
    ReplayUnavailable(u64),
    #[error("event subscriber fell behind the replay buffer")]
    Lagged,
    #[error("event stream is already terminal")]
    PublishAfterTerminal,
}

#[derive(Clone)]
pub struct EventBus {
    state: Arc<Mutex<State>>,
}

struct State {
    command_id: String,
    next_sequence: u64,
    capacity: usize,
    history: VecDeque<CommandEvent>,
    sender: broadcast::Sender<CommandEvent>,
    terminal: bool,
}

pub struct EventReceiver {
    replay: VecDeque<CommandEvent>,
    receiver: broadcast::Receiver<CommandEvent>,
    closed: bool,
}

impl EventBus {
    pub fn new(command_id: impl Into<String>) -> Self {
        Self::with_capacity(command_id, DEFAULT_HISTORY_CAPACITY)
    }

    pub fn with_capacity(command_id: impl Into<String>, capacity: usize) -> Self {
        let capacity = capacity.max(1);
        let (sender, _) = broadcast::channel(capacity);
        Self {
            state: Arc::new(Mutex::new(State {
                command_id: command_id.into(),
                next_sequence: 0,
                capacity,
                history: VecDeque::with_capacity(capacity),
                sender,
                terminal: false,
            })),
        }
    }

    pub fn publish(&self, kind: CommandEventKind) -> Result<CommandEvent, EventError> {
        let mut state = self.state.lock().expect("event bus lock poisoned");
        if state.terminal {
            return Err(EventError::PublishAfterTerminal);
        }

        state.next_sequence += 1;
        let event = CommandEvent {
            command_id: state.command_id.clone(),
            sequence: state.next_sequence,
            kind,
        };
        if state.history.len() == state.capacity {
            state.history.pop_front();
        }
        state.history.push_back(event.clone());
        if event.kind.is_terminal() {
            state.terminal = true;
        }
        let _ = state.sender.send(event.clone());
        Ok(event)
    }

    pub fn subscribe(&self, after_sequence: u64) -> Result<EventReceiver, EventError> {
        let state = self.state.lock().expect("event bus lock poisoned");
        if let Some(oldest) = state.history.front().map(|event| event.sequence) {
            if after_sequence.saturating_add(1) < oldest {
                return Err(EventError::ReplayUnavailable(after_sequence));
            }
        }

        Ok(EventReceiver {
            replay: state
                .history
                .iter()
                .filter(|event| event.sequence > after_sequence)
                .cloned()
                .collect(),
            receiver: state.sender.subscribe(),
            closed: false,
        })
    }
}

impl EventReceiver {
    pub async fn recv(&mut self) -> Result<Option<CommandEvent>, EventError> {
        if self.closed {
            return Ok(None);
        }
        if let Some(event) = self.replay.pop_front() {
            if event.kind.is_terminal() {
                self.closed = true;
            }
            return Ok(Some(event));
        }

        match self.receiver.recv().await {
            Ok(event) => {
                if event.kind.is_terminal() {
                    self.closed = true;
                }
                Ok(Some(event))
            }
            Err(broadcast::error::RecvError::Lagged(_)) => Err(EventError::Lagged),
            Err(broadcast::error::RecvError::Closed) => Ok(None),
        }
    }
}
