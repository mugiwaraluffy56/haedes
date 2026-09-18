import type { SandboxState } from '@haedes/sdk';

const stateLabels: Record<SandboxState, string> = {
  requested: 'Requested',
  provisioning: 'Provisioning',
  starting: 'Starting',
  running: 'Running',
  snapshotting: 'Snapshotting',
  stopping: 'Stopping',
  stopped: 'Stopped',
  failed: 'Failed',
  destroyed: 'Destroyed',
};

export function StateBadge({ state }: { state: SandboxState }) {
  return <span className={`state-badge state-${state}`}><i />{stateLabels[state]}</span>;
}

export function stateLabel(state: SandboxState): string {
  return stateLabels[state];
}
