import type { CommandEvent, CommandResult } from './types.js';

export interface SseFrame {
  id?: string;
  event?: string;
  data: string;
}

export class SseProtocolError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'SseProtocolError';
  }
}

export async function* parseSse(response: Response, signal?: AbortSignal): AsyncGenerator<SseFrame> {
  if (!response.body) {
    throw new SseProtocolError('SSE response did not include a body.');
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let frame: SseFrame = { data: '' };
  let hasData = false;

  const consumeLine = (line: string): SseFrame | undefined => {
    if (line === '') {
      if (!hasData) {
        return undefined;
      }
      const complete = frame;
      frame = { data: '' };
      hasData = false;
      return complete;
    }
    if (line.startsWith(':')) {
      return undefined;
    }
    const separator = line.indexOf(':');
    const field = separator === -1 ? line : line.slice(0, separator);
    const value = separator === -1 ? '' : line.slice(separator + 1).replace(/^ /, '');
    if (field === 'id') frame.id = value;
    if (field === 'event') frame.event = value;
    if (field === 'data') {
      frame.data += `${frame.data === '' ? '' : '\n'}${value}`;
      hasData = true;
    }
    return undefined;
  };

  const abort = () => {
    void reader.cancel(signal?.reason);
  };
  signal?.addEventListener('abort', abort, { once: true });

  try {
    while (true) {
      if (signal?.aborted) throw signal.reason ?? new DOMException('The operation was aborted.', 'AbortError');
      const { done, value } = await reader.read();
      buffer += decoder.decode(value ?? new Uint8Array(), { stream: !done });
      const lines = buffer.split(/\r?\n/);
      buffer = lines.pop() ?? '';
      for (const line of lines) {
        const complete = consumeLine(line);
        if (complete) yield complete;
      }
      if (done) break;
    }

    if (buffer !== '') {
      const complete = consumeLine(buffer);
      if (complete) yield complete;
    }
    const complete = consumeLine('');
    if (complete) yield complete;
  } finally {
    signal?.removeEventListener('abort', abort);
    reader.releaseLock();
  }
}

export function eventFromFrame(frame: SseFrame): CommandEvent {
  let parsed: unknown;
  try {
    parsed = JSON.parse(frame.data);
  } catch {
    throw new SseProtocolError('SSE command event data was not valid JSON.');
  }

  if (!parsed || typeof parsed !== 'object') {
    throw new SseProtocolError('SSE command event data must be a JSON object.');
  }

  const payload = parsed as Record<string, unknown>;
  const type = normalizeEventType(payload.type, frame.event);
  if (typeof payload.commandId !== 'string' || typeof payload.at !== 'string') {
    throw new SseProtocolError(`SSE ${type} event is missing commandId or at.`);
  }

  switch (type) {
    case 'started':
      return { type, commandId: payload.commandId, at: payload.at };
    case 'stdout':
    case 'stderr':
      if (typeof payload.data !== 'string') {
        throw new SseProtocolError(`SSE ${type} event is missing data.`);
      }
      return { type, commandId: payload.commandId, data: payload.data, at: payload.at };
    case 'completed':
      if (!payload.result || typeof payload.result !== 'object') {
        throw new SseProtocolError('SSE completed event is missing result.');
      }
      return { type, commandId: payload.commandId, result: payload.result as CommandResult, at: payload.at };
    case 'failed':
      if (typeof payload.code !== 'string' || typeof payload.message !== 'string') {
        throw new SseProtocolError('SSE failed event is missing code or message.');
      }
      return { type, commandId: payload.commandId, code: payload.code, message: payload.message, at: payload.at };
  }
}

function normalizeEventType(payloadType: unknown, frameType?: string): CommandEvent['type'] {
  const candidate = typeof payloadType === 'string' ? payloadType : frameType;
  switch (candidate) {
    case 'started':
    case 'CommandStartedEvent':
      return 'started';
    case 'stdout':
    case 'stderr':
      return candidate;
    case 'completed':
    case 'CommandCompletedEvent':
      return 'completed';
    case 'failed':
    case 'CommandFailedEvent':
      return 'failed';
    default:
      throw new SseProtocolError(`Unsupported SSE command event type: ${String(candidate)}.`);
  }
}

export function isTerminalEvent(event: CommandEvent): boolean {
  return event.type === 'completed' || event.type === 'failed';
}
