import assert from 'node:assert/strict';
import { test } from 'node:test';
import { authenticate, knownUser } from '../src/auth.js';

test('accepts the fixture user with the correct password', () => {
  assert.equal(knownUser('agent'), true);
  assert.equal(authenticate('agent', 'correct-horse-battery-staple'), true);
});

test('rejects an incorrect password', () => {
  assert.equal(authenticate('agent', 'wrong-password'), false);
});
