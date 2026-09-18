import { commandPayload } from './commands.js';
import { normalizeConfig, type NormalizedConfig, type SandboxClientConfig } from './config.js';
import { SandboxError } from './errors.js';
import { eventFromFrame, isTerminalEvent, parseSse, SseProtocolError } from './events.js';
import { SandboxCollection } from './sandbox.js';
import type { CommandEvent, CommandResult, ExecOptions } from './types.js';

export class SandboxClient {
  readonly sandboxes: SandboxCollection;
  private readonly config: NormalizedConfig;

  constructor(config: SandboxClientConfig) {
    this.config = normalizeConfig(config);
    this.sandboxes = new SandboxCollection(this);
  }

  async requestJson<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await this.request(path, init);
    if (!response.ok) {
      throw await SandboxError.fromResponse(response);
    }
    return (await response.json()) as T;
  }

  async requestBytes(path: string, init: RequestInit = {}): Promise<Uint8Array> {
    const response = await this.request(path, init);
    if (!response.ok) {
      throw await SandboxError.fromResponse(response);
    }
    return new Uint8Array(await response.arrayBuffer());
  }

  async requestEmpty(path: string, init: RequestInit = {}): Promise<void> {
    const response = await this.request(path, init);
    if (!response.ok) {
      throw await SandboxError.fromResponse(response);
    }
  }

  async *streamCommandEvents(
    sandboxId: string,
    commandId: string,
    options: Pick<ExecOptions, 'signal'> & { lastEventId?: string } = {},
  ): AsyncGenerator<CommandEvent> {
    let lastEventId = options.lastEventId;
    let reconnectAttempted = false;

    while (true) {
      options.signal?.throwIfAborted();
      const headers = new Headers({ Accept: 'text/event-stream' });
      if (lastEventId !== undefined) headers.set('Last-Event-ID', lastEventId);
      let response: Response;
      try {
        response = await this.request(`/v1/sandboxes/${encodeURIComponent(sandboxId)}/commands/${encodeURIComponent(commandId)}/events`, {
          headers,
          signal: options.signal,
        });
        if (!response.ok) {
          throw await SandboxError.fromResponse(response);
        }
        for await (const frame of parseSse(response, options.signal)) {
          if (frame.id !== undefined) lastEventId = frame.id;
          const event = eventFromFrame(frame);
          yield event;
          if (isTerminalEvent(event)) return;
        }
      } catch (error) {
        if (options.signal?.aborted) {
          throw options.signal.reason ?? error;
        }
        if (error instanceof SandboxError || error instanceof SseProtocolError) throw error;
        if (reconnectAttempted) throw error;
      }
      if (reconnectAttempted) return;
      reconnectAttempted = true;
    }
  }

  private async request(path: string, init: RequestInit = {}): Promise<Response> {
    const controller = new AbortController();
    let timedOut = false;
    const timeout = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, this.config.requestTimeoutMs);
    const abortExternal = () => controller.abort();
    init.signal?.addEventListener('abort', abortExternal, { once: true });
    const headers = new Headers(init.headers);
    headers.set('Authorization', `Bearer ${this.config.apiKey}`);
    if (init.body !== undefined && init.body !== null && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }
    try {
      return await this.config.fetch(`${this.config.baseUrl}${path}`, {
        ...init,
        headers,
        signal: controller.signal,
      });
    } catch (error) {
      if (timedOut && error instanceof DOMException && error.name === 'AbortError') {
        throw new SandboxError({ code: 'request_timeout', message: 'Sandbox API request timed out.', status: 408 });
      }
      throw error;
    } finally {
      clearTimeout(timeout);
      init.signal?.removeEventListener('abort', abortExternal);
    }
  }

  async createCommand(sandboxId: string, command: string, options: ExecOptions = {}): Promise<CommandResult> {
    return this.requestJson<CommandResult>(`/v1/sandboxes/${encodeURIComponent(sandboxId)}/commands`, {
      method: 'POST',
      body: JSON.stringify(commandPayload(command, options)),
      signal: options.signal,
    });
  }

  static idempotencyKey(): string {
    const cryptoObject = globalThis.crypto as Crypto | undefined;
    if (cryptoObject?.randomUUID) return cryptoObject.randomUUID();
    return `sdk-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  }
}
