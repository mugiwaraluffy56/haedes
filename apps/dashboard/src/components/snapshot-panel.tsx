'use client';

import { useState } from 'react';
import type { Sandbox, SnapshotMetadata } from '@haedes/sdk';
import { createSnapshot, restoreSnapshot, toDashboardError, type DashboardError } from '../lib/api-client';
import { ErrorPanel } from './error-panel';

export function SnapshotPanel({ sandbox, onChanged }: { sandbox: Sandbox; onChanged: () => Promise<void> }) {
  const [latest, setLatest] = useState<SnapshotMetadata | null>(null);
  const [busy, setBusy] = useState<'snapshot' | 'restore' | null>(null);
  const [error, setError] = useState<DashboardError | null>(null);
  const snapshotId = latest?.id ?? sandbox.snapshotIds.at(-1);

  async function saveSnapshot() {
    setBusy('snapshot');
    setError(null);
    try { setLatest(await createSnapshot(sandbox.id)); await onChanged(); } catch (cause) { setError(toDashboardError(cause)); } finally { setBusy(null); }
  }

  async function restoreLatest() {
    if (!snapshotId) return;
    setBusy('restore');
    setError(null);
    try {
      const restored = await restoreSnapshot(snapshotId);
      window.location.assign(`/dashboard/sandboxes/${encodeURIComponent(restored.id)}`);
    } catch (cause) { setError(toDashboardError(cause)); setBusy(null); }
  }

  return <section className="activity-panel snapshot-panel"><div className="activity-heading"><div><span className="section-kicker">Workspace persistence</span><h2>Snapshots</h2></div><span className="snapshot-count">{sandbox.snapshotIds.length} saved</span></div>{error && <ErrorPanel error={error} />}<div className="snapshot-summary"><div className="snapshot-orb">◌</div><div><strong>{latest?.state ?? (snapshotId ? 'available' : 'No snapshot yet')}</strong><p>{snapshotId ? snapshotId : 'Save the workspace before releasing compute.'}</p></div></div><div className="snapshot-actions"><button type="button" onClick={() => void saveSnapshot()} disabled={busy !== null || sandbox.state !== 'running'}>{busy === 'snapshot' ? 'Saving…' : 'Save workspace'}</button><button type="button" className="button-muted" onClick={() => void restoreLatest()} disabled={busy !== null || !snapshotId}>{busy === 'restore' ? 'Restoring…' : 'Restore into new sandbox'}</button></div><p className="panel-note">Snapshots preserve the workspace while the temporary AWS computer can be destroyed.</p></section>;
}
