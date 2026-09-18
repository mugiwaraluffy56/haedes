'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import type { Sandbox } from '@haedes/sdk';
import { ErrorPanel } from './error-panel';
import { StateBadge } from './state-badge';
import { getSandbox, isNonTerminal, toDashboardError, type DashboardError } from '../lib/api-client';

function formatDate(value: string): string { return new Intl.DateTimeFormat('en', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)); }
function lifecycleStatus(state: Sandbox['state']): { label: string; detail: string } {
  if (state === 'running' || state === 'snapshotting') return { label: 'Task active', detail: 'The private AWS computer is available to the agent.' };
  if (state === 'stopping') return { label: 'Releasing compute', detail: 'Cleanup is in progress and new work is rejected.' };
  if (state === 'stopped' || state === 'destroyed') return { label: 'Compute released', detail: 'The AWS computer is no longer active.' };
  if (state === 'failed') return { label: 'Requires attention', detail: 'The control plane retained a failure state for investigation.' };
  return { label: 'Provisioning computer', detail: 'The control plane is preparing a private AWS computer.' };
}

export function SandboxDetail({ id }: { id: string }) {
  const [sandbox, setSandbox] = useState<Sandbox | null>(null);
  const [error, setError] = useState<DashboardError | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try { setSandbox(await getSandbox(id)); setError(null); } catch (cause) { setError(toDashboardError(cause)); } finally { setLoading(false); }
  }, [id]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    if (!sandbox || !isNonTerminal(sandbox.state)) return;
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [sandbox, refresh]);

  const status = sandbox ? lifecycleStatus(sandbox.state) : null;

  return (
    <main className="dashboard-shell detail-shell">
      <header className="dashboard-nav"><a className="dashboard-wordmark" href="/"><span>h</span>haedes</a><div className="dashboard-nav-right"><span className="api-indicator"><i /> Live API</span><Link href="/sandboxes">All sandboxes ↗</Link></div></header>
      <div className="detail-content">
        <Link className="back-link" href="/sandboxes">← Back to execution fleet</Link>
        {loading && <div className="detail-loading"><div /><div /><div /></div>}
        {error && <ErrorPanel error={error} />}
        {sandbox && status && <>
          <section className="detail-heading"><div><p className="dashboard-eyebrow">Sandbox detail</p><h1>{sandbox.id}</h1><p className="detail-repository">{sandbox.repository?.url ?? 'Untitled workspace'}</p></div><StateBadge state={sandbox.state} /></section>
          <section className="status-banner"><div className={`status-orb orb-${sandbox.state}`} /><div><span className="status-label">AWS EXECUTION STATUS</span><h2>{status.label}</h2><p>{status.detail}</p></div><span className="status-state">{sandbox.state}</span></section>
          <div className="detail-grid">
            <DetailCard title="Lifecycle timestamps"><DetailRow label="Created" value={formatDate(sandbox.createdAt)} /><DetailRow label="Expires" value={formatDate(sandbox.expiresAt)} /><DetailRow label="Last activity" value={formatDate(sandbox.lastActivityAt)} /></DetailCard>
            <DetailCard title="Resource allocation"><DetailRow label="CPU" value={`${sandbox.config.cpuMillis} millicores`} /><DetailRow label="Memory" value={`${sandbox.config.memoryMiB} MiB`} /><DetailRow label="Storage" value={`${sandbox.config.storageGiB} GiB`} /></DetailCard>
            <DetailCard title="Workspace state"><DetailRow label="Current command" value={sandbox.currentCommandId ?? 'No command running'} mono={Boolean(sandbox.currentCommandId)} /><DetailRow label="Snapshots" value={String(sandbox.snapshotIds.length)} /><DetailRow label="Default timeout" value={`${sandbox.config.defaultCommandTimeoutSeconds}s`} /></DetailCard>
            <DetailCard title="Runtime configuration"><DetailRow label="Image" value={sandbox.config.image} mono /><DetailRow label="Environment" value={`${Object.keys(sandbox.config.environment ?? {}).length} variables`} /><DetailRow label="Max lifetime" value={`${sandbox.config.maxLifetimeSeconds}s`} /></DetailCard>
          </div>
          {sandbox.repository && <section className="repository-panel"><span className="section-kicker">Repository workspace</span><strong>{sandbox.repository.url}</strong><span>Provider: {sandbox.repository.provider} · Path: {sandbox.repository.path}{sandbox.repository.ref ? ` · Ref: ${sandbox.repository.ref}` : ''}</span></section>}
        </>}
      </div>
      <footer className="dashboard-footer"><span>haedes dashboard</span><span>Lifecycle is owned by the control plane.</span></footer>
    </main>
  );
}

function DetailCard({ title, children }: { title: string; children: React.ReactNode }) { return <section className="detail-card"><span className="section-kicker">{title}</span>{children}</section>; }
function DetailRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) { return <div className="detail-row"><span>{label}</span><strong className={mono ? 'mono' : ''}>{value}</strong></div>; }
