import { SandboxClient } from '@haedes/sdk';
import { WorkspaceSession, type AgentSandboxSession } from './workspace-session.js';

/**
 * Provider-neutral construction boundary. A real agent owns the decisions and
 * calls the returned session; this adapter never invokes a model or plans work.
 */
export function createAgentSandboxSession(client: SandboxClient): AgentSandboxSession {
  return new WorkspaceSession(client);
}
