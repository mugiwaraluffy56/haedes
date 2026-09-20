'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import type { Page, Sandbox } from '@haedes/sdk';
import { ErrorPanel } from './error-panel';
import { SandboxCard } from './sandbox-card';
import { StateBadge } from './state-badge';
import { isNonTerminal, listSandboxes, toDashboardError, type DashboardError } from '../lib/api-client';

export function SandboxList() {
  const [page, setPage] = useState<Page<Sandbox> | null>(null);
  const [error, setError] = useState<DashboardError | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const refresh = useCallback(async () => {
    setRefreshing(true);
    try { setPage(await listSandboxes()); setError(null); } catch (cause) { setError(toDashboardError(cause)); } finally { setLoading(false); setRefreshing(false); }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => { if (!page?.items.some((sandbox) => isNonTerminal(sandbox.state))) return; const timer = window.setInterval(() => void refresh(), 5000); return () => window.clearInterval(timer); }, [page, refresh]);

  const items = page?.items ?? [];
  const activeCount = items.filter((sandbox) => isNonTerminal(sandbox.state)).length;
  const runningCount = items.filter((sandbox) => sandbox.state === 'running').length;
  const attentionCount = items.filter((sandbox) => ['failed', 'provisioning', 'starting'].includes(sandbox.state)).length;

  return <main className="dashboard-shell">
    <header className="dashboard-nav shell"><Link className="dashboard-wordmark" href="/"><Image src="/haedes-logo-white.svg" alt="haedes" width={20} height={20} priority /></Link><div className="dashboard-nav-right"><span className="api-indicator"><i /> Live API</span><Link href="/">Back home</Link></div></header>
    <div className="dashboard-content shell">
      <section className="dashboard-hero"><div><p className="dashboard-eyebrow">Dashboard / Sandboxes</p><h1>Execution fleet</h1><p className="dashboard-lede">Monitor temporary computers, lifecycle state, and workspace access from one operational surface.</p></div><div className="hero-actions"><span className="demo-label">{refreshing ? 'Syncing' : 'Live API'}</span><button type="button" onClick={() => void refresh()} disabled={refreshing}>{refreshing ? 'Refreshing…' : 'Refresh'}</button></div></section>
      <section className="metric-grid" aria-label="Fleet metrics"><Metric label="Total" value={page ? items.length : '—'} detail="sandbox records" /><Metric label="Active" value={page ? activeCount : '—'} detail={`${runningCount} running now`} accent /><Metric label="Attention" value={page ? attentionCount : '—'} detail="needs observation" /><Metric label="Refresh" value={activeCount > 0 ? '5s' : 'off'} detail="polling interval" /></section>
    </div>
    {error && <ErrorPanel error={error} />}
    <div className="dashboard-content shell dashboard-body-grid">
      <section className="sandbox-section"><div className="section-toolbar"><div><span className="section-kicker">All sandboxes</span><h2>Sandbox fleet</h2></div><span className="record-count">{page ? `${items.length} records` : 'Loading'}</span></div>{loading ? <div className="loading-grid" aria-label="Loading sandboxes"><div /><div /><div /></div> : items.length ? <div className="sandbox-grid">{items.map((sandbox) => <SandboxCard key={sandbox.id} sandbox={sandbox} />)}</div> : <EmptyState />}</section>
      <aside className="monitoring-stack"><ActivityPanel activeCount={activeCount} attentionCount={attentionCount} /><HealthPanel /></aside>
    </div>
    <footer className="dashboard-footer shell"><span>haedes dashboard</span><span>Data comes from the configured /v1 API. No local mock state.</span></footer>
  </main>;
}

function Metric({ label, value, detail, accent = false }: { label: string; value: string | number; detail: string; accent?: boolean }) { return <div className="metric-card"><span>{label}</span><strong className={accent ? 'metric-accent' : ''}>{value}</strong><small>{detail}</small></div>; }
function ActivityPanel({ activeCount, attentionCount }: { activeCount: number; attentionCount: number }) { return <section className="monitoring-panel"><div className="monitoring-heading"><h2>Recent activity</h2><span>LIVE</span></div><Activity color="running" title="Fleet heartbeat" detail={`${activeCount} active lifecycle${activeCount === 1 ? '' : 's'}`} time="now" /><Activity color="attention" title="Attention queue" detail={`${attentionCount} sandbox${attentionCount === 1 ? '' : 'es'} need observation`} time="live" /><Activity color="muted" title="Workspace contract" detail="Control plane owns lifecycle and cleanup" time="/v1" /></section>; }
function Activity({ color, title, detail, time }: { color: string; title: string; detail: string; time: string }) { return <div className="monitoring-row"><i className={`activity-dot activity-${color}`} /><div><strong>{title}</strong><small>{detail}</small></div><time>{time}</time></div>; }
function HealthPanel() { return <section className="monitoring-panel health-panel"><h2>Workspace health</h2><HealthRow label="CONTROL PLANE" value="CONNECTED" accent /><HealthRow label="RUNTIME" value="RUST / READY" /><HealthRow label="WORKSPACE" value="ISOLATED" /><HealthRow label="CLEANUP" value="AUTOMATIC" /></section>; }
function HealthRow({ label, value, accent = false }: { label: string; value: string; accent?: boolean }) { return <div className="health-row"><span>{label}</span><strong className={accent ? 'metric-accent' : ''}>{value}</strong></div>; }

function EmptyState() { return <div className="empty-state"><div className="empty-orbit">◌</div><h3>No sandboxes yet</h3><p>When an agent requests a computer, its lifecycle will appear here.</p><StateBadge state="requested" /></div>; }
