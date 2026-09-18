'use client';

import { useCallback, useEffect, useState } from 'react';
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
    try {
      setPage(await listSandboxes());
      setError(null);
    } catch (cause) {
      setError(toDashboardError(cause));
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  useEffect(() => {
    if (!page?.items.some((sandbox) => isNonTerminal(sandbox.state))) return;
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [page, refresh]);

  const activeCount = page?.items.filter((sandbox) => isNonTerminal(sandbox.state)).length ?? 0;

  return (
    <main className="dashboard-shell">
      <header className="dashboard-nav"><a className="dashboard-wordmark" href="/"><span>h</span>haedes</a><div className="dashboard-nav-right"><span className="api-indicator"><i /> Live API</span><a href="/">Back to haedes ↗</a></div></header>
      <section className="dashboard-hero">
        <div><p className="dashboard-eyebrow">Human observability / sandbox fleet</p><h1>Your computers,<br /><em>in view.</em></h1><p className="dashboard-lede">Watch agent work move from request to running computer to clean release.</p></div>
        <div className="fleet-summary"><span className="summary-label">SANDBOXES</span><strong>{page?.items.length ?? '—'}</strong><span className="summary-live">{activeCount} active lifecycle{activeCount === 1 ? '' : 's'}</span></div>
      </section>
      {error && <ErrorPanel error={error} />}
      <section className="sandbox-section">
        <div className="section-toolbar"><div><span className="section-kicker">All sandboxes</span><h2>Execution fleet</h2></div><div className="toolbar-actions">{activeCount > 0 && <span className="polling-label"><i /> Polling active sandboxes</span>}<button type="button" onClick={() => void refresh()} disabled={refreshing}>{refreshing ? 'Refreshing…' : 'Refresh ↻'}</button></div></div>
        {loading ? <div className="loading-grid" aria-label="Loading sandboxes"><div /><div /><div /></div> : page?.items.length ? <div className="sandbox-grid">{page.items.map((sandbox) => <SandboxCard key={sandbox.id} sandbox={sandbox} />)}</div> : <EmptyState />}
      </section>
      <footer className="dashboard-footer"><span>haedes dashboard</span><span>Data comes from the configured /v1 API. No local mock state.</span></footer>
    </main>
  );
}

function EmptyState() {
  return <div className="empty-state"><div className="empty-orbit">◌</div><h3>No sandboxes yet</h3><p>When an agent requests a computer, its lifecycle will appear here.</p><StateBadge state="requested" /></div>;
}
