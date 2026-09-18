import Link from 'next/link';
import type { Sandbox } from '@haedes/sdk';
import { StateBadge } from './state-badge';

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('en', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }).format(new Date(value));
}

function executionStatus(state: Sandbox['state']): string {
  if (state === 'running' || state === 'snapshotting') return 'Task active';
  if (state === 'stopping') return 'Releasing compute';
  if (state === 'stopped' || state === 'destroyed') return 'Compute released';
  if (state === 'failed') return 'Requires attention';
  return 'Provisioning computer';
}

export function SandboxCard({ sandbox }: { sandbox: Sandbox }) {
  return (
    <Link className="sandbox-card" href={`/sandboxes/${encodeURIComponent(sandbox.id)}`}>
      <div className="sandbox-card-top"><span className="sandbox-id">{sandbox.id}</span><StateBadge state={sandbox.state} /></div>
      <div className="sandbox-card-title">{sandbox.repository?.url ?? 'Untitled workspace'}</div>
      <div className="sandbox-card-meta"><span>Created {formatDate(sandbox.createdAt)}</span><span>Expires {formatDate(sandbox.expiresAt)}</span></div>
      <div className="sandbox-card-bottom"><span><i className={`execution-dot execution-${sandbox.state}`} />{executionStatus(sandbox.state)}</span><span className="card-arrow">↗</span></div>
    </Link>
  );
}
