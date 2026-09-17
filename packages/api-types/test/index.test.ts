import assert from 'node:assert/strict';
import { parseApiError } from '../src/index.js';

const validError = {
  error: {
    code: 'sandbox_not_running',
    message: 'Sandbox is not running.',
    requestId: 'req_01HZZ',
    details: { state: 'stopped' },
  },
};

assert.deepEqual(parseApiError(validError), validError.error);
assert.equal(parseApiError({ error: { code: 'bad' } }), undefined);
assert.equal(parseApiError({ error: { code: 'bad', message: 'bad', requestId: 'req_01HZZ', details: [] } }), undefined);

console.log('api error parsing checks passed');
