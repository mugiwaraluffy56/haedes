# haedes

## 1. Product Summary

haedes gives AI agents temporary cloud computers running on AWS where they can safely perform real work.

An agent can request a Linux environment, clone a repository, inspect files, install dependencies, execute commands, modify code, build projects, run tests, save its workspace, and destroy the environment when the task is finished.

Developers do not need to create servers, configure containers, manage cleanup, or expose their own infrastructure to agent generated code.

The simplest description is:

**Give your AI agent a computer on AWS.**

haedes is accessible at three integration levels:

```text
AI agents, Claude Code, Codex, custom coding agents,
agent frameworks, and internal company agents
                         |
                         v
             Agent native integration layer
                    MCP and adapters
                         |
                         v
              Developer integration layer
                    TypeScript SDK
                         |
                         v
              Universal platform contract
                       HTTP API
                         |
                         v
                    haedes control plane
                         |
                         v
                 Disposable AWS computers
```

The agent-native layer is the most visible product experience. The SDK is the main developer surface for companies building their own agent applications. The versioned HTTP API is the canonical machine contract underneath all of them.

An agent, SDK client, or application requests a sandbox.

haedes creates it.

The agent works inside it.

The agent or application receives the results.

The sandbox disappears when the task is complete.

---

# 2. The Problem

AI agents can generate code and reason about tasks, but useful agents need somewhere to actually perform that work.

Consider a coding agent asked:

**Fix issue 342 in this GitHub repository.**

The agent may need to:

1. Clone the repository

2. Search through the codebase

3. Install project dependencies

4. Run the existing application

5. Execute tests

6. Modify files

7. Compile software

8. Run generated code

9. Inspect failures

10. Retry solutions

11. Produce a final patch

The reasoning happens inside the AI model.

The actual work needs a computer.

Developers should not run arbitrary agent generated commands directly on their application servers or production infrastructure.

Building dedicated infrastructure for this internally also means solving compute provisioning, isolation, cleanup, command execution, filesystem access, logging, storage, timeouts, failures, and scaling.

haedes provides this execution layer as a service. The product is the computer and its lifecycle, not a collection of AWS endpoints.

---

# 3. Product Vision

Every serious AI agent should be able to request temporary compute when it needs to perform work, wherever that agent itself is running.

The experience should feel roughly like:

```text
create AWS sandbox

run command

read files

write files

install dependencies

run tests

save workspace

destroy sandbox
```

The developer should not need to understand the AWS infrastructure underneath.

haedes turns AWS compute into a simple computer abstraction designed specifically for agents.

The platform contract is stable across all integration surfaces:

```text
agent-native tools or adapters  ->  TypeScript SDK  ->  versioned HTTP API
                                                        |
                                                        v
                                      Go control plane -> AWS Fargate computer
```

MCP is the initial agent-native integration for the hackathon. It is a thin adapter over the same sandbox platform; it is not a second backend or the product itself.

---

# 4. Core Product Idea

haedes is not an AI agent.

It is infrastructure that AI agents use.

The AI decides what needs to happen.

haedes gives that AI somewhere safe to execute those decisions.

The product therefore separates intelligence from execution and offers several ways to reach the execution layer.

### Intelligence

Claude

OpenAI

Gemini

Custom models

Coding agents

Research agents

Automation agents

### Execution

haedes

The intelligence thinks.

The sandbox executes.

The agent can be local, hosted by a SaaS product, or embedded in an internal agent application. haedes does not depend internally on Claude, OpenAI, Gemini, or any other model provider. Claude Code, Codex, custom coding agents, and agent frameworks are clients of the platform, subject to the capabilities their current integrations actually expose.

### Integration levels

#### Agent native integration

MCP exposes agent-friendly tools such as `sandbox_create`, `sandbox_get`, `sandbox_list`, `sandbox_exec`, `sandbox_exec_stream`, filesystem operations, `sandbox_snapshot`, `sandbox_restore`, and `sandbox_destroy`. An agent can request a computer as a tool, use it, preserve the workspace, and release the compute.

For the hackathon, one real coding agent client must use this integration. The chosen client and exact setup must be verified in the implementation environment before being described as complete; the core platform remains provider neutral.

#### TypeScript SDK

The SDK gives developers building their own agent applications a product-focused interface to sandboxes. It hides ECS, Fargate, S3, IAM, task ARNs, security groups, and other AWS implementation details.

#### HTTP API

The versioned `/v1` HTTP API is the canonical universal platform contract. It powers the SDK, dashboard, MCP, future agent adapters, customer integrations, and internal tools. HTTP is the stable foundation beneath the product experiences, not the primary product story.

---

# 5. Why AWS Is Core To The Product

AWS is not simply where the website is hosted.

The actual product runs on AWS.

When an agent requests a computer:

1. AWS creates the execution environment.

2. The repository and files exist inside that AWS environment.

3. Commands execute on AWS compute.

4. Logs are collected through AWS.

5. Workspace snapshots are stored on AWS.

6. Sandbox metadata is stored on AWS.

7. Destroying the sandbox terminates AWS compute.

If AWS were removed, the core product would no longer work.

That is the role AWS should have in the product.

---

# 6. Who Uses It

## 6.1 Coding Agent Developers

Teams building agents that fix bugs, implement features, investigate repositories, write tests, or perform engineering tasks.

## 6.2 AI Application Developers

Applications that generate code and need somewhere safe to execute it.

## 6.3 Developer Tool Companies

Products involving repositories, pull requests, code review, debugging, automated testing, or application generation.

## 6.4 Agent Platforms

Agent frameworks that need a standard way to provide their agents with computers.

## 6.5 Education Platforms

Applications where students or AI tutors need to execute code safely.

## 6.6 Research Agents

Agents that need to download repositories, run scripts, transform data, or use command line tools.

## 6.7 Automation Platforms

Applications that dynamically execute scripts or workflows on behalf of users.

---

# 7. Core Product Experience

The primary experience starts in an existing coding agent. The developer gives it a task such as:

```text
Fix the authentication bug in this repository.
```

The agent decides it needs a computer and invokes `sandbox_create` through MCP. A custom agent or agent application can use the same platform through the TypeScript SDK or directly through the HTTP API.

haedes creates a fresh environment on AWS.

The agent receives access to the sandbox.

The agent clones the repository.

The agent installs dependencies.

The agent searches through the code.

The agent runs tests.

A test fails.

The agent reads the error.

The agent modifies the necessary files.

The agent runs the tests again.

The tests pass.

The agent retrieves the resulting changes or leaves them in a saved workspace.

The sandbox is destroyed.

The developer never needed to create or manage the AWS infrastructure themselves. The same flow works when the agent runs locally, in a SaaS product, or inside an internal company system because the remote computer is independent of the agent’s host.

---

# 8. Core Features

## 8.1 Instant AWS Sandboxes

Agents and applications can request temporary Linux computers through MCP, the TypeScript SDK, or the canonical HTTP API.

Every sandbox receives:

A unique sandbox ID

Its own filesystem

Its own running processes

CPU allocation

Memory allocation

Maximum lifetime

Execution history

The developer and agent interact with the sandbox rather than the underlying AWS resources. One sandbox maps to one temporary Fargate task in the initial implementation.

---

# 8.2 Disposable Cloud Computers

Sandboxes are designed to be temporary.

They exist only while useful work is happening.

When the task finishes, the application can destroy the sandbox.

If the application forgets to destroy it, the platform automatically terminates it after the configured lifetime.

The product should make creating and destroying computers feel cheap and normal.

---

# 8.3 Command Execution

Agents can execute Linux commands inside the sandbox.

Examples:

```text
git clone

npm install

npm test

python script.py

cargo test

go test

grep

find

curl
```

Every execution returns:

Output

Errors

Exit status

Execution time

Current state

This is the fundamental feature of the platform.

---

# 8.4 Live Command Streaming

Agents often execute commands that take time.

Developers should be able to see the output while commands run.

Example:

```text
Installing packages...

Building application...

Running tests...

42 tests passed

3 tests failed
```

The dashboard should display this live.

This provides both observability and a strong visual experience during the hackathon demo.

---

# 8.5 Filesystem API

Agents can interact with files inside their AWS sandbox.

The hackathon filesystem contract supports:

Read a file

Write a file

Create a file through write

Delete a file

List directories

Search files by running commands inside the computer

Convenience upload and download helpers can be added later; they are not required for the first public contract.

This allows an agent to treat the sandbox like a real computer.

---

# 8.6 Git Repository Workspaces

Repository based workflows should be one of the strongest experiences.

A developer provides:

```text
github.com/company/project
```

haedes creates the environment and prepares the repository.

The agent can immediately begin working.

The sandbox effectively becomes a temporary cloud development machine dedicated to that task.

---

# 8.7 Dependency Installation

Agents should be able to modify their development environment.

Examples:

```text
npm install

pip install

cargo build

go mod download

apt install
```

This is important because different projects require different environments.

The developer should not have to manually prepare every possible dependency before creating the sandbox.

---

# 8.8 Development Language Support

The default environment should support common development stacks.

Initial support:

Python

JavaScript

TypeScript

Go

Rust

C

C++

Shell

Only one strong base environment is required for the hackathon.

---

# 8.9 Automatic Cleanup

Every sandbox has an expiration time.

The developer can destroy it manually.

Otherwise haedes destroys it automatically once its maximum lifetime is reached.

This prevents forgotten agent jobs from continuously consuming AWS resources.

---

# 8.10 Resource Controls

Developers can control how much compute an agent receives.

Basic options:

CPU

Memory

Maximum runtime

Command timeout

Storage

This lets applications create lightweight environments for simple tasks and larger environments when necessary.

---

# 8.11 Workspace Snapshots

An agent may spend significant time preparing an environment.

For example:

Repository cloned

Dependencies installed

Fifteen files changed

Build completed

Tests partially fixed

Instead of losing that progress, the developer can create a snapshot.

haedes saves the workspace to Amazon S3.

The running sandbox can then be destroyed.

---

# 8.12 Resume From Snapshot

A saved workspace can later be restored into a fresh AWS sandbox.

Example:

Agent A works on a repository.

The model reaches its usage limit.

The workspace is saved.

The original sandbox is destroyed.

Later Agent B starts.

haedes creates a fresh environment.

The workspace is restored from S3.

Agent B continues from the previous state.

The important concept is:

**The computer can disappear while the work survives.**

---

# 8.13 Agent Handoff

Snapshots also allow work to move between agents.

Example:

Agent A investigates the bug.

Agent A modifies the implementation.

The workspace is saved.

Agent B receives a restored environment.

Agent B runs the tests.

Agent C performs a review.

The workspace becomes shared execution context between multiple agents. This is an important future direction enabled by snapshots, but multi-agent execution is not a hackathon requirement.

---

# 8.14 Parallel Sandboxes

Applications can create multiple sandboxes simultaneously.

For example, one agent could attempt solution A while another attempts solution B.

```text
Task
 |
 + Agent A
 |    |
 |    AWS Sandbox A
 |
 + Agent B
      |
      AWS Sandbox B
```

The parent agent can compare the results and continue with the better solution.

This makes the platform useful for more complex agent systems later. It is intentionally deferred until the single-agent vertical slice is reliable.

---

# 8.15 Sandbox Dashboard

The dashboard is the human observability and control surface, not the primary product entry point. It should make agent-driven cloud execution visible.

For every sandbox show:

Sandbox ID

Current status

Repository

Creation time

Runtime

Current command

CPU allocation

Memory allocation

Commands executed

Recent output

Snapshot status

AWS environment status

The dashboard should make it immediately obvious that real compute is being created and destroyed.

---

# 8.16 Live Terminal View

Each sandbox should provide a terminal style activity view.

Example:

```text
$ git clone repository

Cloning...

$ npm install

Packages installed

$ npm test

FAIL auth.test.ts

$ grep authenticate src

$ npm test

PASS
```

This gives developers visibility into what their agent is actually doing.

---

# 8.17 Execution Timeline

The platform should preserve a simple history of actions.

Example:

```text
10:41 sandbox created

10:42 repository cloned

10:43 dependencies installed

10:45 tests executed

10:47 files modified

10:49 tests executed

10:50 snapshot created

10:51 sandbox destroyed
```

This helps developers understand the lifecycle of an autonomous task.

---

# 8.18 Sandbox Templates

Developers should eventually be able to choose prepared environments.

Examples:

Node environment

Python environment

Rust environment

Go environment

Full development environment

Minimal Linux environment

For the hackathon, only a default development environment is required.

---

# 8.19 TypeScript SDK

The platform should expose a TypeScript SDK as an important developer integration surface. It hides AWS complexity and exposes the computer abstraction.

Conceptually:

```typescript
const sandbox = await Sandbox.create();

await sandbox.exec("git clone ...");

await sandbox.exec("npm install");

const result = await sandbox.exec("npm test");

await sandbox.destroy();
```

The developer does not deal with:

EC2

ECS

Fargate

IAM

VPC configuration

Containers

S3 APIs

CloudWatch configuration

They interact with a sandbox.

---

# 8.20 Agent-native MCP integration

The initial agent-native integration is a thin MCP server over the sandbox platform. Its conceptual tools are:

```text
sandbox_create
sandbox_get
sandbox_list
sandbox_exec
sandbox_exec_stream
sandbox_read_file
sandbox_write_file
sandbox_list_files
sandbox_delete_file
sandbox_snapshot
sandbox_restore
sandbox_destroy
```

The MCP adapter validates tool inputs, calls the SDK or the shared HTTP client, returns structured agent-friendly results, exposes sandbox and command IDs, streams command events where supported, and maps typed platform errors into useful tool errors. It never calls ECS directly, persists lifecycle state independently, receives or exposes AWS credentials, or implements another lifecycle state machine.

MCP is replaceable. Future adapters for other agent hosts can converge on the same HTTP contract without changing the Go control plane or Rust runtime.

# 8.21 Canonical HTTP platform contract

The `/v1` API remains the universal platform contract. It retains versioning, lifecycle semantics, idempotency, SSE command events, filesystem routes, snapshot routes, authentication, request IDs, and typed error envelopes. The dashboard, SDK, MCP server, future adapters, customer integrations, and internal tools are all clients of this contract.

The API must not leak ECS task definitions, task ARNs, S3 APIs, IAM roles, security groups, or Fargate configuration into the developer-facing sandbox abstraction.

# 9. Primary Use Cases

## 9.1 Coding Agents

An AI coding agent receives a software engineering task.

It invokes the MCP integration to request a sandbox. An agent application built by a developer can use the same flow through the TypeScript SDK or directly through the HTTP API.

The repository is prepared.

The agent edits code and tests its work inside AWS.

---

# 9.2 GitHub Issue Agents

Input:

```text
Fix issue 421.
```

The agent receives an AWS environment containing the project.

It investigates the issue.

It edits the repository.

It runs tests.

It produces the resulting changes.

---

# 9.3 Pull Request Review Agents

A review agent checks out a pull request inside a temporary AWS sandbox.

It can:

Build the project

Run tests

Execute static analysis

Inspect dependencies

Attempt reproduction

Review runtime behavior

The environment disappears after the review.

---

# 9.4 AI Generated Code Execution

An application asks a model to generate Python or JavaScript.

Instead of executing that code on the application's server, it sends it to an AWS sandbox.

The sandbox executes it.

The output is returned.

The sandbox is destroyed.

---

# 9.5 Automated Debugging

The user provides:

A repository

An error

An expected behavior

The agent creates an AWS sandbox.

It reproduces the problem.

Tests possible fixes.

Runs tests.

Returns the final change.

---

# 9.6 Application Generation

An agent generates an application inside the sandbox.

It can:

Create files

Install packages

Build the application

Execute it

Run tests

Return the complete project

---

# 9.7 Data Analysis Agents

A user provides a dataset.

The agent creates an AWS sandbox.

It installs required libraries.

Runs analysis.

Creates output files.

Returns results.

The execution environment is then destroyed.

---

# 9.8 Research Agents

Research agents can use the sandbox when browser access is not enough.

They can:

Clone repositories

Download datasets

Execute scripts

Run command line tools

Transform files

Compile programs

Perform analysis

---

# 9.9 Education Platforms

Student code can execute inside temporary AWS sandboxes instead of the main education platform server.

Every submission gets its own disposable environment.

---

# 9.10 Multi Agent Engineering

A coordinator agent can create multiple AWS sandboxes.

Different agents attempt different approaches.

Each sandbox remains isolated.

The coordinator receives the results.

The winning implementation can then continue.

This is a future facing use case but can become an impressive extension after the main product works.

---

# 10. AWS Product Architecture

The product should be AWS native from the beginning.

```text
Claude Code / Codex / custom agent / agent framework
                        |
                        v
              MCP or agent adapter
                        |
                        v
                 TypeScript SDK
                  when useful
                        |
                        v
                  HTTP API /v1
                        |
                        v
               Go Control Plane
                        |
                +-------+-------+
                |               |
                v               v
          AWS orchestration   metadata
                |
                v
          ECS / Fargate task
                |
                v
          Rust Sandbox Runtime
                |
                v
             /workspace
```

All integration surfaces converge on the same sandbox platform. MCP may delegate to the TypeScript SDK or use a shared HTTP client directly; either way it does not own domain logic. The Go control plane remains the source of truth for lifecycle, authentication, authorization, AWS orchestration, metadata, and event fan-out. One sandbox maps to one temporary Fargate task in the initial implementation. The Rust runtime owns processes and `/workspace`.

Supporting services:

```text
Amazon S3
Workspace snapshots

Amazon DynamoDB
Sandbox metadata

Amazon CloudWatch
Logs and monitoring

Amazon ECR
Sandbox images

Amazon VPC
Sandbox networking
```

---

# 11. Technology Stack

## Frontend

Next.js

TypeScript

Used for:

Dashboard

Sandbox management

Live terminal output

Execution timeline

Snapshot management

The dashboard is a human observability and control surface for agent activity. It is not the primary product entry point.

# 11.10 Agent-native integration

MCP is the one agent protocol in the hackathon scope. It exposes sandbox tools and delegates to the existing platform contract. The architecture is provider neutral: Claude Code, Codex, custom coding agents, and agent frameworks are conceptual clients, and no vendor is treated as an internal dependency. The exact client used in the demo must be verified before claiming support for it.

---

# 11.1 Control Plane

Go

Responsible for:

Public API

Creating sandboxes

Destroying sandboxes

Tracking lifecycle

Communicating with AWS

Managing snapshot operations

Managing sandbox metadata

Routing requests to active environments

Go is used because the control plane primarily deals with cloud orchestration, network services, APIs, and concurrent infrastructure operations.

---

# 11.2 Sandbox Runtime

Rust

Every sandbox runs a lightweight Rust service.

The Rust runtime handles:

Command execution

Process management

Filesystem access

Live output streaming

Command timeouts

Workspace preparation

Snapshot preparation

Environment cleanup

Rust fits this component because it operates close to processes and the filesystem.

---

# 11.3 Sandbox Compute

Amazon ECS with AWS Fargate.

Each sandbox becomes its own temporary Fargate task.

The developer does not interact with ECS or Fargate directly.

haedes converts these AWS primitives into a simple computer API.

---

# 11.4 Storage

Amazon S3.

Used for:

Workspace snapshots

Uploaded project files

Saved sandbox artifacts

Exported results

---

# 11.5 Metadata

Amazon DynamoDB.

Stores:

Sandbox ID

Sandbox state

Creation time

Expiration time

Repository

Resource configuration

Snapshot references

Task information

---

# 11.6 Logs

Amazon CloudWatch.

Used for:

Sandbox runtime logs

Control plane logs

Errors

Infrastructure debugging

Health monitoring

---

# 11.7 Images

Amazon ECR.

Stores the base sandbox environment used when new AWS sandboxes are created.

---

# 11.8 Networking

Amazon VPC.

Used to separate and control communication between the platform, control plane, and sandbox environments.

---

# 11.9 SDK

TypeScript first.

Python later.

The SDK is one developer integration surface. It should hide AWS entirely from the application developer and must not expose ECS task definitions, task ARNs, S3 APIs, IAM roles, security groups, or Fargate configuration.

# 11.11 HTTP API

The versioned `/v1` HTTP API is the canonical universal platform contract. It powers the SDK, dashboard, MCP, future agent adapters, customer integrations, and internal tools. Keep its lifecycle semantics, idempotency, SSE command events, filesystem operations, snapshot routes, authentication, request IDs, and error model stable.

---

# 12. What Local Development Means

Developers building haedes itself may use Docker locally for development and testing.

That is only an internal development workflow.

It is not part of the user facing product.

Production sandboxes are created on AWS.

The public abstraction remains:

```text
Agent
  |
  v
MCP, SDK, or HTTP API
  |
  v
Temporary AWS computer
```

There is no user facing local sandbox option in the hackathon product.

This keeps the product focused and makes AWS central to the experience.

---

# 13. Killer Hackathon Demo

The demonstration should begin inside the chosen real coding agent and tell one complete story: an existing agent suddenly gained access to disposable AWS computers.

The user gives the agent:

```text
Fix the authentication bug in this repository.
```

The agent decides it needs a computer and invokes `sandbox_create` through MCP. The exact client setup is shown only after it has been verified in the demo environment.

The dashboard shows the sandbox moving through:

```text
requested
provisioning
starting
running
```

Briefly show the ECS task appearing in AWS. The agent then:

- invokes `sandbox_exec("git clone ...")` inside the Fargate computer;
- installs dependencies;
- runs the tests while the dashboard streams stdout and stderr;
- observes the authentication test failure;
- reads the relevant files and modifies the implementation;
- runs the test suite again and shows it passing;
- invokes `sandbox_snapshot` and briefly shows the S3 object;
- invokes `sandbox_destroy` and shows the Fargate task stopping;
- creates a completely new sandbox;
- invokes `sandbox_restore` for the previous snapshot;
- runs the tests again and shows that they still pass; and
- destroys the second sandbox.

The video should make the underlying command IDs, sandbox IDs, lifecycle states, and AWS evidence legible without turning the dashboard into the primary actor.

This one demonstration proves:

AWS compute provisioning

Agent integration

Command execution

Repository support

Live output

Filesystem modification

Isolation

Snapshots

Destruction

Restoration

Cloud execution

The story is deliberately one real coding-agent flow. Multi-agent execution, multiple agent protocols, and broad framework support remain future directions.

---

# 14. AWS Visibility During The Demo

Because the hackathon requires visible AWS usage, the video should explicitly show it.

Do not only show the product dashboard.

Briefly show:

Amazon ECS task appearing when the sandbox is created.

CloudWatch logs appearing while commands execute.

S3 snapshot appearing when the workspace is saved.

ECS task stopping when the sandbox is destroyed.

Then immediately return to the product.

The message should be:

**Our interface is simple, but everything underneath is actually happening on AWS.**

---

# 15. What Makes It More Than Docker

Docker may exist underneath development environments, but Docker itself is not the product.

A developer using haedes receives:

Remote cloud execution

Compute provisioning

Lifecycle management

Command APIs

Filesystem APIs

Live command output

Automatic cleanup

Resource controls

Workspace persistence

Snapshots

Recovery

SDK integration

Observability

AWS infrastructure

The developer never needs to manage the infrastructure directly.

The abstraction is not:

```text
container
```

It is:

```text
computer for my agent
```

---

# 16. Product Positioning

Primary positioning:

**Give your AI agent a computer on AWS.**

Alternative:

**AWS cloud computers for AI agents.**

Alternative:

**Safe cloud execution for autonomous agents.**

Alternative:

**The AWS execution layer for AI agents.**

The clearest hackathon pitch is:

**haedes gives your AI agent a disposable cloud computer on AWS.**

The HTTP API remains the stable universal contract beneath the agent-native and SDK experiences.

---

# 17. Hackathon Scope

The goal is not to recreate every sandbox platform.

The goal is to make one experience work extremely well.

## Must Build

AWS sandbox creation

AWS sandbox destruction

Command execution

Live command output

Filesystem operations

Git repository cloning

Dependency installation

Resource timeout

Workspace snapshots using S3

Workspace restoration

Basic dashboard

Execution history

TypeScript SDK

Canonical versioned `/v1` HTTP API with authentication, lifecycle semantics, idempotency, SSE command events, filesystem routes, snapshot routes, and typed errors

One MCP agent-native integration over that existing platform contract

One real coding agent using the MCP integration, subject to verified client compatibility

Visible AWS infrastructure in the demo

## Build If Time Allows

Parallel sandboxes

Agent handoff

Sandbox templates

Artifact downloads

Resource configuration

## Do Not Prioritize

GPU environments

Browser automation

Kubernetes

Custom images

Billing

Enterprise authentication

Organization management

Multiple regions

Marketplace

Advanced networking controls

Complex autoscaling

Local product mode

Multiple MCP implementations or provider-specific Claude Code, Codex, or framework plugins

A generic agent framework or a custom agent loop as the primary demo

The product needs depth in its main execution flow rather than dozens of unfinished features.

---

# 18. Questions Strict Judges May Ask

## Why does an AI agent need its own computer?

Because agents that perform engineering work need to execute commands, install dependencies, manipulate files, compile projects, and run tests.

The AI provides intelligence.

The sandbox provides execution.

## Why AWS?

The product dynamically creates computers for agents.

AWS already provides the compute, storage, networking, databases, container infrastructure, and observability needed to operate those environments.

AWS therefore becomes the underlying execution platform rather than simply a hosting provider.

## Is AWS just hosting your backend?

No.

Every agent sandbox itself runs on AWS.

When an agent executes:

```text
npm test
```

that command actually runs inside an AWS sandbox.

When the sandbox is destroyed, the AWS compute is terminated.

When the workspace is saved, it is persisted using S3.

## Why not run the agent on the user's laptop?

Many agent products operate remotely.

The user's laptop may be offline.

The task may take a long time.

Multiple agents may need simultaneous computers.

Generated commands may also be unsafe to execute directly on the user's personal machine.

Remote sandboxes separate execution from the user's device.

## Why not run the agent on your backend server?

An agent may execute unpredictable code.

Sandboxing separates that code from the application's control plane.

Destroying the sandbox also removes the entire temporary environment after the task ends.

## Why not just use Docker?

Docker is an infrastructure primitive.

haedes provides a developer product around execution.

The application does not manage containers.

It requests computers.

The service handles provisioning, access, lifecycle, execution, storage, logs, cleanup, and restoration.

## Why not E2B?

E2B demonstrates that agent sandbox infrastructure is a real product category.

haedes focuses on building an AWS native execution platform where disposable cloud computers, workspace persistence, and coding agent workflows are first class concepts.

The hackathon project does not need to claim that it already replaces established products.

It needs to demonstrate a meaningful product and working implementation.

## Why save the workspace instead of keeping the sandbox running?

Keeping compute alive wastes resources.

Snapshots allow the platform to terminate the expensive running environment while preserving useful work.

Another environment can later restore the workspace.

## What happens if the agent crashes?

The sandbox can remain active until timeout.

The application can reconnect.

If a workspace snapshot exists, another environment can also restore the saved work.

## What happens if the AI provider reaches its limit?

The workspace can be saved.

The current agent can stop.

Another model or agent can later restore the environment and continue.

## What happens if the agent executes dangerous code?

The code executes inside the sandbox rather than the control plane.

The environment has a limited lifetime and limited resources.

The entire environment can be destroyed.

## What happens if a program runs forever?

Commands have execution limits.

Sandboxes also have maximum lifetimes.

Expired environments are terminated automatically.

## Can multiple agents run simultaneously?

Yes.

Each agent can receive an independent AWS sandbox.

Later versions can also allow agents to share saved workspace state.

## Is this only useful for coding agents?

No.

Any agent needing a shell, filesystem, packages, programs, or temporary compute can use it.

Coding agents are simply the initial target because the value is easiest to demonstrate.

## What have you actually built instead of simply connecting AWS services?

The product layer.

The sandbox API.

The SDK.

The Rust execution runtime.

Command streaming.

Filesystem controls.

Sandbox lifecycle.

Workspace snapshot and restoration.

Agent integration.

The thin MCP adapter that lets one real coding agent use those capabilities.

Dashboard.

AWS provides infrastructure primitives.

The product turns them into agent execution infrastructure.

---

# 19. Success Criteria

The project succeeds if a judge watches the demonstration and understands these points without needing an explanation afterward:

An AI agent needed a computer.

Our agent-native integration, SDK, or HTTP API created that computer on AWS.

The agent performed real work inside it.

The agent could execute arbitrary development commands.

The developer did not manage AWS manually.

The environment was disposable.

The work could survive after compute disappeared.

A fresh AWS environment could continue that work.

AWS was fundamental to the product.

---

# 20. Final Product Statement

**haedes is execution infrastructure for AI agents. Developers give their agents temporary Linux computers running on AWS through MCP, the TypeScript SDK, or the canonical HTTP API, allowing them to clone repositories, install dependencies, execute commands, modify files, run tests, save their work, and destroy the environment when the task is complete.**

The AI agent is the brain.

haedes is the computer it works on.
