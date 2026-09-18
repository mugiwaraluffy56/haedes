export interface CommandRun {
  id?: string;
  command: string;
  status: 'running' | 'completed' | 'failed' | 'error';
  startedAt?: string;
  finishedAt?: string;
  durationMs?: number;
  detail?: string;
}

function formatTime(value?: string): string {
  if (!value) return '—';
  return new Intl.DateTimeFormat('en', { hour: 'numeric', minute: '2-digit', second: '2-digit' }).format(new Date(value));
}

function formatDuration(value?: number): string {
  if (value === undefined) return 'duration —';
  if (value < 1000) return `${value}ms`;
  const seconds = value / 1000;
  return `duration ${seconds < 10 ? seconds.toFixed(1) : Math.round(seconds)}s`;
}

export function ExecutionTimeline({ runs }: { runs: CommandRun[] }) {
  return <section className="activity-panel timeline-panel"><div className="activity-heading"><div><span className="section-kicker">Execution history</span><h2>Command timeline</h2></div><span className="activity-count">{runs.length} {runs.length === 1 ? 'run' : 'runs'}</span></div>{runs.length === 0 ? <p className="activity-empty">Commands started by the agent will appear here during this dashboard session.</p> : <div className="timeline-list">{[...runs].reverse().map((run, index) => <div className="timeline-item" key={`${run.id ?? run.command}-${index}`}><span className={`timeline-marker timeline-${run.status}`} /><div className="timeline-main"><div><strong>{run.command}</strong><span className={`timeline-status timeline-status-${run.status}`}>{run.status}</span></div><p>{run.detail ?? 'Command submitted through the public API.'}</p></div><time>{formatTime(run.startedAt)} · {formatDuration(run.durationMs)}</time></div>)}</div>}</section>;
}
