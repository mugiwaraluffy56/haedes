'use client';

import { useMemo, useState } from 'react';
import type { CommandEvent } from '@haedes/sdk';
import { ErrorPanel } from './error-panel';
import { getSandboxHandle, toDashboardError, type DashboardError } from '../lib/api-client';
import type { CommandRun } from './execution-timeline';

function eventText(event: CommandEvent): string {
  if (event.type === 'stdout') return event.data;
  if (event.type === 'stderr') return `[stderr] ${event.data}`;
  if (event.type === 'started') return `Started ${event.commandId}`;
  if (event.type === 'completed') return `Process completed with exit code ${event.result.exitCode ?? 'null'}`;
  if (event.type === 'failed') return `${event.code}: ${event.message}`;
  return 'The command stream ended unexpectedly.';
}

export function Terminal({ sandboxId, enabled, onRunComplete }: { sandboxId: string; enabled: boolean; onRunComplete: (run: CommandRun) => void }) {
  const [command, setCommand] = useState('npm test');
  const [events, setEvents] = useState<CommandEvent[]>([]);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<DashboardError | null>(null);
  const output = useMemo(() => events.map(eventText).join('\n'), [events]);

  async function runCommand(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!command.trim() || running || !enabled) return;
    const submitted = command.trim();
    const controller = new AbortController();
    setEvents([]);
    setError(null);
    setRunning(true);
    const collected: CommandEvent[] = [];
    const startedAt = new Date().toISOString();
    try {
      const handle = await getSandboxHandle(sandboxId);
      for await (const nextEvent of handle.execStream(submitted, { signal: controller.signal })) {
        collected.push(nextEvent);
        setEvents((current) => [...current, nextEvent]);
      }
      const terminal = collected.at(-1);
      const completedAt = new Date().toISOString();
      const commandStartedAt = terminal?.type === 'completed'
        ? terminal.result.startedAt
        : collected.find((nextEvent) => nextEvent.type === 'started')?.at ?? startedAt;
      const commandFinishedAt = terminal?.type === 'completed' ? terminal.result.finishedAt ?? completedAt : completedAt;
      const run: CommandRun = {
        id: terminal?.type === 'completed' || terminal?.type === 'failed' ? terminal.commandId : undefined,
        command: submitted,
        status: terminal?.type === 'completed' ? 'completed' : terminal?.type === 'failed' ? 'failed' : 'error',
        startedAt: commandStartedAt,
        finishedAt: commandFinishedAt,
        durationMs: Math.max(0, Date.parse(commandFinishedAt) - Date.parse(commandStartedAt)),
        detail: terminal?.type === 'completed' ? `Exit code ${terminal.result.exitCode ?? 'null'}` : terminal?.type === 'failed' ? terminal.message : 'The stream ended before a terminal event.',
      };
      onRunComplete(run);
    } catch (cause) {
      const dashboardError = toDashboardError(cause);
      setError(dashboardError);
      const finishedAt = new Date().toISOString();
      onRunComplete({ command: submitted, status: 'error', startedAt, finishedAt, durationMs: Math.max(0, Date.parse(finishedAt) - Date.parse(startedAt)), detail: dashboardError.message });
    } finally {
      setRunning(false);
    }
  }

  return <section className="activity-panel terminal-panel"><div className="activity-heading"><div><span className="section-kicker">Live command stream</span><h2>Agent terminal</h2></div><span className={`stream-status ${running ? 'stream-live' : ''}`}><i />{running ? 'Streaming' : 'Ready'}</span></div><div className="terminal-screen" aria-live="polite"><div className="terminal-line terminal-muted">haedes /workspace</div>{output ? <pre>{output}</pre> : <div className="terminal-placeholder">Run a command to watch stdout and stderr arrive from the sandbox.</div>}</div>{error && <ErrorPanel error={error} />}<form className="terminal-form" onSubmit={runCommand}><span className="terminal-prompt">$</span><input aria-label="Command" value={command} onChange={(event) => setCommand(event.target.value)} disabled={running || !enabled} placeholder="Enter a command" /><button type="submit" disabled={running || !enabled || !command.trim()}>{running ? 'Running…' : 'Run command ↵'}</button></form><p className="terminal-note">The SDK resumes one dropped SSE connection from the last event ID. Output remains ordered.</p></section>;
}
