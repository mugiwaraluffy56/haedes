const dashboardUrl = process.env.NEXT_PUBLIC_DASHBOARD_URL ?? '/dashboard';

const sdkExample = `const sandbox = await client.sandboxes.create({
  image: 'haedes-sandbox-dev:dev',
});

for await (const event of sandbox.execStream('npm test')) {
  if (event.type === 'stdout') process.stdout.write(event.data);
}

const snapshot = await sandbox.snapshot();
await sandbox.destroy();`;

const lifecycle = [
  ['01', 'Request', 'An agent asks for a workspace through MCP, the SDK, or HTTP.'],
  ['02', 'Provision', 'The control plane starts one private Fargate computer.'],
  ['03', 'Work', 'Commands, files, tests, and live output stay inside /workspace.'],
  ['04', 'Preserve', 'A compressed workspace snapshot keeps useful work after compute ends.'],
];

export default function Home() {
  return (
    <main>
      <nav className="nav shell" aria-label="Main navigation">
        <a className="wordmark" href="#top" aria-label="haedes home">
          <span className="wordmark-mark">h</span>
          <span>haedes</span>
        </a>
        <div className="nav-links">
          <a href="#platform">Platform</a>
          <a href="#how-it-works">How it works</a>
          <a href="#developers">Developers</a>
          <a className="nav-cta" href={dashboardUrl}>Open dashboard <span aria-hidden="true">↗</span></a>
        </div>
      </nav>

      <section className="hero shell" id="top">
        <div className="hero-copy">
          <p className="eyebrow"><span className="eyebrow-dot" /> Agent execution infrastructure</p>
          <h1>Give your AI agent <em>a computer on AWS.</em></h1>
          <p className="hero-lede">
            haedes gives agents a temporary Linux computer to clone repositories, install dependencies,
            run commands, change files, and keep working until the job is done.
          </p>
          <div className="hero-actions">
            <a className="button button-primary" href={dashboardUrl}>See the dashboard <span aria-hidden="true">↗</span></a>
            <a className="text-link" href="#developers">Read the SDK example <span aria-hidden="true">↓</span></a>
          </div>
          <div className="hero-proof">
            <span className="status-pulse" />
            <span>One sandbox</span><b>·</b><span>One private task</span><b>·</b><span>Zero lasting infrastructure</span>
          </div>
        </div>

        <div className="hero-visual" aria-label="A sandbox lifecycle from request to a running workspace">
          <div className="visual-glow" />
          <div className="visual-card visual-card-back" />
          <div className="visual-card visual-card-main">
            <div className="visual-topline"><span className="mini-label">SANDBOX / LIVE</span><span className="live-pill"><i /> RUNNING</span></div>
            <div className="visual-title">sbx_01HZX9V4</div>
            <div className="visual-command"><span className="prompt">$</span><span>npm test</span><span className="cursor" /></div>
            <div className="visual-output">
              <div><span className="output-muted">›</span> Resolving workspace dependencies...</div>
              <div><span className="output-muted">›</span> Running 42 tests</div>
              <div className="output-success">✓ 42 passed in 3.18s</div>
            </div>
            <div className="visual-footer"><span>CPU <strong>512m</strong></span><span>MEM <strong>1 GiB</strong></span><span>TTL <strong>09:42</strong></span></div>
          </div>
          <div className="floating-chip chip-task"><span className="chip-icon">⌁</span><span><small>AWS COMPUTE</small><strong>Fargate task</strong></span></div>
          <div className="floating-chip chip-snapshot"><span className="chip-icon">◌</span><span><small>WORKSPACE</small><strong>Snapshot ready</strong></span></div>
        </div>
      </section>

      <section className="section shell brain-section" id="platform">
        <div className="section-intro narrow">
          <p className="eyebrow">The boundary</p>
          <h2>The agent is the brain.<br /><span>haedes is the computer.</span></h2>
          <p>Keep intelligence wherever it belongs. Give execution a clean, isolated place to happen.</p>
        </div>
        <div className="boundary-grid">
          <article className="boundary-card brain-card">
            <span className="card-index">01 / INTELLIGENCE</span>
            <div className="card-symbol symbol-brain">✦</div>
            <h3>Your agent decides</h3>
            <p>Reason about the task, choose the next command, inspect failures, and decide when the work is complete.</p>
            <div className="card-tags"><span>Model</span><span>Planner</span><span>Agent loop</span></div>
          </article>
          <div className="boundary-connector" aria-hidden="true"><span>hands off</span><b>→</b></div>
          <article className="boundary-card computer-card">
            <span className="card-index">02 / EXECUTION</span>
            <div className="card-symbol symbol-computer">⌘</div>
            <h3>haedes does the work</h3>
            <p>Provision a real computer, run processes, stream output, preserve the workspace, and clean up automatically.</p>
            <div className="card-tags"><span>Linux</span><span>Workspace</span><span>AWS</span></div>
          </article>
        </div>
      </section>

      <section className="section platform-section" id="how-it-works">
        <div className="shell">
          <div className="section-intro split-intro">
            <div><p className="eyebrow">One platform, three entry points</p><h2>Meet the computer<br /><span>where you already work.</span></h2></div>
            <p>Every surface converges on the same versioned platform contract. Start with the interface that fits your agent or application.</p>
          </div>
          <div className="entry-grid">
            <article className="entry-card entry-featured"><span className="entry-number">01</span><div className="entry-icon">⌁</div><h3>MCP</h3><p>Agent-native tools for creating a sandbox, executing commands, handling files, and saving work.</p><a href="#developers">Explore the flow <span aria-hidden="true">↗</span></a></article>
            <article className="entry-card"><span className="entry-number">02</span><div className="entry-icon">‹›</div><h3>TypeScript SDK</h3><p>A product-focused developer surface that keeps AWS implementation details out of your application.</p><a href="#developers">See the SDK <span aria-hidden="true">↗</span></a></article>
            <article className="entry-card"><span className="entry-number">03</span><div className="entry-icon">/v1</div><h3>HTTP API</h3><p>The stable contract underneath the dashboard, SDK, MCP adapter, and future integrations.</p><a href="#architecture">View the architecture <span aria-hidden="true">↗</span></a></article>
          </div>
        </div>
      </section>

      <section className="section lifecycle-section">
        <div className="shell">
          <div className="section-intro narrow"><p className="eyebrow">A computer with a lifecycle</p><h2>Useful work in.<br /><span>Compute out.</span></h2><p>Sandboxes are temporary by design. Workspace state can survive the computer that created it.</p></div>
          <div className="lifecycle-list">
            {lifecycle.map(([number, title, description]) => <div className="lifecycle-row" key={number}><span className="lifecycle-number">{number}</span><h3>{title}</h3><p>{description}</p><span className="lifecycle-arrow" aria-hidden="true">↗</span></div>)}
          </div>
        </div>
      </section>

      <section className="section architecture-section" id="architecture">
        <div className="shell architecture-grid">
          <div className="architecture-copy"><p className="eyebrow">Under the abstraction</p><h2>Simple for the agent.<br /><span>Real on AWS.</span></h2><p>The product vocabulary stays focused on sandboxes, commands, files, workspaces, and snapshots. Underneath, each request follows a deliberate ownership boundary.</p><a className="text-link" href={dashboardUrl}>Watch a sandbox come alive <span aria-hidden="true">↗</span></a></div>
          <div className="architecture-map">
            <div className="map-line map-line-1" /><div className="map-line map-line-2" /><div className="map-line map-line-3" />
            <div className="map-node map-node-agent"><small>AGENT SURFACE</small><strong>MCP · SDK · HTTP</strong></div>
            <div className="map-node map-node-control"><small>GO CONTROL PLANE</small><strong>Lifecycle + orchestration</strong></div>
            <div className="map-node map-node-runtime"><small>RUST RUNTIME</small><strong>Processes + /workspace</strong></div>
            <div className="map-node map-node-aws"><small>AWS</small><strong>Private Fargate task</strong></div>
          </div>
        </div>
      </section>

      <section className="section developer-section" id="developers">
        <div className="shell developer-grid">
          <div className="developer-copy"><p className="eyebrow">Built for real work</p><h2>Give an agent<br /><span>somewhere to go.</span></h2><p>Create a sandbox, use it like a computer, and save the workspace when the useful part is done.</p><a className="button button-primary" href={dashboardUrl}>Open the dashboard <span aria-hidden="true">↗</span></a></div>
          <div className="code-window"><div className="code-top"><span className="window-dots"><i /><i /><i /></span><span>agent.ts</span><span className="code-status">@haedes/sdk</span></div><pre><code>{sdkExample}</code></pre></div>
        </div>
      </section>

      <footer className="footer shell"><a className="wordmark" href="#top"><span className="wordmark-mark">h</span><span>haedes</span></a><span>Give your AI agent a computer on AWS.</span><a href="#top">Back to top ↑</a></footer>
    </main>
  );
}
