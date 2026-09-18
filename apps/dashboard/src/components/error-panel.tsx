import type { DashboardError } from '../lib/api-client';

export function ErrorPanel({ error }: { error: DashboardError }) {
  return (
    <div className="error-panel" role="alert">
      <div className="error-icon">!</div>
      <div>
        <strong>API request failed</strong>
        <p>{error.message}</p>
        {error.requestId ? <code>Request ID: {error.requestId}</code> : <small>Check the dashboard API configuration and try again.</small>}
      </div>
    </div>
  );
}
