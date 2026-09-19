'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useCallback, useEffect, useState } from 'react';
import type { Sandbox } from '@haedes/sdk';
import { AwsStatus } from './aws-status';
import { ErrorPanel } from './error-panel';
import { ExecutionTimeline, type CommandRun } from './execution-timeline';
import { SnapshotPanel } from './snapshot-panel';
import { StateBadge } from './state-badge';
import { Terminal } from './terminal';
import { MotionToggle } from './motion';
import { destroySandbox, getSandbox, isNonTerminal, toDashboardError, type DashboardError } from '../lib/api-client';

function formatDate(value: string): string { return new Intl.DateTimeFormat('en', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)); }

export function SandboxDetail({ id }: { id: string }) {
  const [sandbox, setSandbox] = useState<Sandbox | null>(null);
  const [error, setError] = useState<DashboardError | null>(null);
  const [actionError, setActionError] = useState<DashboardError | null>(null);
  const [loading, setLoading] = useState(true);
  const [destroying, setDestroying] = useState(false);
  const [runs, setRuns] = useState<CommandRun[]>([]);

  const refresh = useCallback(async () => {
    try { setSandbox(await getSandbox(id)); setError(null); } catch (cause) { setError(toDashboardError(cause)); } finally { setLoading(false); }
  }, [id]);

  async function destroy() {
    if (!sandbox || destroying || !window.confirm(`Destroy ${sandbox.id}? New work will be rejected.`)) return;
    setDestroying(true);
    setActionError(null);
    try { await destroySandbox(sandbox.id); await refresh(); } catch (cause) { setActionError(toDashboardError(cause)); } finally { setDestroying(false); }
  }

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    if (!sandbox || !isNonTerminal(sandbox.state)) return;
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [sandbox, refresh]);

  return (
    <main className="dashboard-shell detail-shell">
      <header className="dashboard-nav"><Link className="dashboard-wordmark" href="/"><Image src="/haedes-logo-white.svg" alt="" width={27} height={27} priority />haedes</Link><div className="dashboard-nav-right"><span className="api-indicator"><i /> Live API</span><MotionToggle /><Link href="/sandboxes">All sandboxes ↗</Link></div></header>
      <div className="detail-content">
        <Link className="back-link" href="/sandboxes">← Back to execution fleet</Link>
        {loading && <div className="detail-loading"><div /><div /><div /></div>}
        {error && <ErrorPanel error={error} />}
        {sandbox && <>
          <section className="detail-heading"><div><p className="dashboard-eyebrow">Sandbox detail</p><h1>{sandbox.id}</h1><p className="detail-repository">{sandbox.repository?.url ?? 'Untitled workspace'}</p></div><div className="detail-heading-actions"><StateBadge state={sandbox.state} /><button className="destroy-button" type="button" onClick={() => void destroy()} disabled={destroying || sandbox.state === 'destroyed'}>{destroying ? 'Destroying…' : 'Destroy sandbox'}</button></div></section>
          <AwsStatus state={sandbox.state} />
          {actionError && <ErrorPanel error={actionError} />}
          <div className="activity-layout"><Terminal sandboxId={sandbox.id} enabled={sandbox.state === 'running'} onRunComplete={(run) => setRuns((current) => [...current, run])} /><ExecutionTimeline runs={runs} /></div>
          <div className="controls-layout"><SnapshotPanel sandbox={sandbox} onChanged={refresh} /></div>
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
