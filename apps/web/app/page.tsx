import Image from 'next/image';

const dashboardUrl = process.env.NEXT_PUBLIC_DASHBOARD_URL ?? '/dashboard';

const lifecycle = [
  ['1', 'Request a sandbox', 'An agent calls MCP, the TypeScript SDK, or the /v1 API with the workspace it needs.'],
  ['2', 'Do the work', 'The Go control plane provisions private Fargate compute while the runtime owns processes and /workspace.'],
  ['3', 'Preserve or destroy', 'Stream output, capture a snapshot when useful, and clean up temporary compute when the task is done.'],
];

const guardrails = [
  ['◌', 'Fast feedback loops', 'Agents can install dependencies, run tests, inspect failures, and iterate on a real Linux filesystem.'],
  ['□', 'Bounded execution', 'Command limits, timeouts, workspace isolation, and automatic cleanup keep compute predictable.'],
  ['⌁', 'One lifecycle owner', 'The control plane owns state transitions and AWS orchestration; adapters stay thin.'],
  ['◉', 'Human visibility', 'The dashboard shows sandbox state, commands, snapshots, errors, and requests without becoming the agent’s brain.'],
  ['▱', 'Provider boundaries', 'AWS details stay behind interfaces and adapters while the public API speaks in sandboxes, commands, files, and snapshots.'],
  ['↗', 'Designed for agents', 'Start in the coding agent, then use the dashboard as the human control surface when you need to see what is happening.'],
];

function FeatureCard({ number, title, description, kind }: { number: string; title: string; description: string; kind: 'agent' | 'workspace' | 'contract' }) {
  return (
    <article className="feature-card">
      <span className="feature-number">{number}</span>
      <div className={`feature-illustration feature-illustration-${kind}`} aria-hidden="true">
        {kind === 'agent' && <><span>$ haedes run</span><strong>&quot;provision workspace&quot;</strong><span>&gt; Creating sandbox…</span><span>&gt; Installing dependencies</span><div className="feature-progress"><i /></div><small>step 4 of 6 <b>80%</b></small><em>&gt; Running tests_</em></>}
        {kind === 'workspace' && <><span>workspace/</span><span>├── <b>src/</b></span><span>│   ├── <b>runtime.rs</b></span><span>│   └── workspace/</span><span>├── <b>api/</b></span><span>│   └── routes.rs</span><span>└── package.json</span><small>workspace ready · <b>isolated</b></small></>}
        {kind === 'contract' && <><div className="contract-heading"><i /> sandbox-042 <b>Live</b></div><span>POST /v1/sandboxes · active</span><span>3 files changed · <b>+42</b> −8</span><span>◉ CI passing · 12/12</span><span>◉ Workspace isolated</span><span>◉ Ready to snapshot</span></>}
      </div>
      <div className="feature-copy"><h3>{title}</h3><p>{description}</p></div>
    </article>
  );
}

export default function Home() {
  return (
    <main>
      <nav className="nav shell" aria-label="Main navigation">
        <a className="logo-link" href="#top" aria-label="haedes home"><Image src="/haedes-logo-white.svg" alt="haedes" width={20} height={20} priority /></a>
        <div className="nav-links">
          <a href="#platform">Platform</a><a href="#interfaces">Interfaces</a><a href="#lifecycle">Lifecycle</a><a href="#guardrails">Guardrails</a><a href="https://github.com/mugiwaraluffy56/haedes">GitHub</a>
        </div>
        <a className="nav-home" href={dashboardUrl}>Open dashboard</a>
      </nav>

      <section className="hero shell" id="top">
        <div className="hero-copy">
          <p className="eyebrow">HUMAN OBSERVABILITY / SANDBOX FLEET</p>
          <h1>Give your agent a computer.</h1>
          <p className="hero-lede">Haedes gives agents a temporary Linux computer to clone code, install dependencies, run commands, edit files, run tests, and keep useful work isolated until the job is done.</p>
          <div className="hero-actions"><a className="button button-primary" href={dashboardUrl}>Open dashboard <span>↗</span></a><a className="text-link" href="#interfaces">Read the interfaces <span>↓</span></a></div>
        </div>
        <div className="hero-visual" aria-label="Haedes sandbox execution preview">
          <div className="visual-card visual-card-back" />
          <div className="visual-card visual-card-main"><div className="visual-topline"><span>sbx_demo_01</span><b><i /> RUNNING</b></div><h2>Execution fleet</h2><div className="visual-command"><span>$</span> npm test <i /></div><div className="visual-output"><p>› Resolving workspace dependencies…</p><p>› Running 42 tests</p><strong>✓ 42 passed in 3.18s</strong></div><div className="visual-footer"><span>CPU <b>512m</b></span><span>MEM <b>1 GiB</b></span><span>TTL <b>09:42</b></span></div></div>
          <span className="visual-chip chip-task">AWS COMPUTE <b>Fargate task</b></span><span className="visual-chip chip-snapshot">WORKSPACE <b>Snapshot ready</b></span>
        </div>
      </section>

      <section className="section shell" id="platform"><div className="section-heading"><p className="eyebrow">PLATFORM</p><h2>MCP, SDK, or HTTP.<br /><span>Use the surface that fits your agent.</span></h2></div><div className="feature-grid" id="interfaces"><FeatureCard number="0.1" title="MCP for the agent loop" description="Create a sandbox, execute bounded commands, handle files, stream output, and preserve work through a thin agent-facing adapter." kind="agent" /><FeatureCard number="0.2" title="The computer is the product" description="Each sandbox is an isolated AWS task with a real filesystem, processes, dependencies, and a lifecycle the control plane owns." kind="workspace" /><FeatureCard number="0.3" title="MCP, SDK, and HTTP together" description="The TypeScript SDK and MCP server stay thin over the versioned /v1 API, so agents and applications share one platform boundary." kind="contract" /></div></section>

      <section className="section lifecycle-section" id="lifecycle"><div className="shell"><div className="section-heading"><p className="eyebrow">LIFECYCLE</p><h2>Request. Run. Preserve.</h2></div><div className="lifecycle-grid">{lifecycle.map(([number, title, description]) => <article className="lifecycle-card" key={number}><span className="lifecycle-number">{number}</span><div className="lifecycle-preview"><span>SANDBOX</span><i /><small>PRIVATE WORKSPACE <b>READY</b></small></div><h3>{title}</h3><p>{description}</p></article>)}</div></div></section>

      <section className="section guardrails-section" id="guardrails"><div className="shell"><div className="section-heading"><p className="eyebrow">GUARDRAILS</p><h2>Temporary compute.<br /><span>Isolated workspaces. Clear boundaries.</span></h2></div><div className="guardrail-grid">{guardrails.map(([icon, title, description]) => <article className="guardrail-card" key={title}><span className="guardrail-icon">{icon}</span><h3>{title}</h3><p>{description}</p></article>)}</div></div></section>

      <footer className="footer shell"><div><a className="logo-link" href="#top"><Image src="/haedes-logo-white.svg" alt="haedes" width={20} height={20} /></a><p>Temporary Linux computers for AI agents.</p><small>Built for agent execution.</small></div><div className="footer-links"><div><strong>Navigation</strong><a href="#platform">Platform</a><a href="#interfaces">Interfaces</a><a href="#lifecycle">Lifecycle</a><a href="#guardrails">Guardrails</a><a href="https://github.com/mugiwaraluffy56/haedes">GitHub</a></div><div><strong>Links</strong><a href="https://github.com/mugiwaraluffy56/haedes">Repository</a><a href={dashboardUrl}>Dashboard</a><a href="https://github.com/mugiwaraluffy56/haedes/issues">Issues</a></div></div></footer>
    </main>
  );
}
