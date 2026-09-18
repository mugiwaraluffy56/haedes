import type { CommandEvent } from './types.js';

interface SseFrame {
  id?: string;
  event?: string;
  data: string;
}

export async function* parseSse(response: Response): AsyncGenerator<SseFrame> {
  if (!response.body) {
    throw new Error('SSE response did not include a body.');
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let frame: SseFrame = { data: '' };

  const consumeLine = (line: string): SseFrame | undefined => {
    if (line === '') {
      if (frame.data === '') {
        return undefined;
      }
      const complete = frame;
      frame = { data: '' };
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
    if (field === 'data') frame.data += `${frame.data === '' ? '' : '\n'}${value}`;
    return undefined;
  };

  while (true) {
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
}

export function eventFromFrame(frame: SseFrame): CommandEvent {
  const payload = JSON.parse(frame.data) as Record<string, unknown>;
  if (frame.event) {
    payload.type = frame.event;
  }
  return payload as CommandEvent;
}

export function isTerminalEvent(event: CommandEvent): boolean {
  return event.type === 'completed' || event.type === 'failed';
}
