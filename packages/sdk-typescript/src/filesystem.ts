import type { FileContent } from './types.js';

export function fileBody(content: FileContent): Uint8Array {
  if (typeof content === 'string') {
    return new TextEncoder().encode(content);
  }
  return content;
}
