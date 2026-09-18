import type { Sandbox } from '@haedes/sdk';

function lifecycleStatus(state: Sandbox['state']): { label: string; detail: string } {
  if (state === 'running' || state === 'snapshotting') return { label: 'Task active', detail: 'The private AWS computer is available to the agent.' };
  if (state === 'stopping') return { label: 'Releasing compute', detail: 'Cleanup is in progress and new work is rejected.' };
  if (state === 'stopped' || state === 'destroyed') return { label: 'Compute released', detail: 'The AWS computer is no longer active.' };
  if (state === 'failed') return { label: 'Requires attention', detail: 'The control plane retained a failure state for investigation.' };
  return { label: 'Provisioning computer', detail: 'The control plane is preparing a private AWS computer.' };
}

export function AwsStatus({ state }: { state: Sandbox['state'] }) {
  const status = lifecycleStatus(state);
  return <section className="status-banner"><div className={`status-orb orb-${state}`} /><div><span className="status-label">AWS EXECUTION STATUS</span><h2>{status.label}</h2><p>{status.detail}</p></div><span className="status-state">{state}</span></section>;
}
