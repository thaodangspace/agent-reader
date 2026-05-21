import assert from 'node:assert/strict';
import test from 'node:test';

import { computeDisplayGroups } from './displayGroups.js';

test('groups consecutive assistant tool-only messages into one display message', () => {
  const groups = computeDisplayGroups([
    {
      id: 'a1',
      role: 'assistant',
      rawText: '',
      thinking: '',
      timestamp: '04:41 PM',
      toolCalls: [{ id: 't1', name: 'bash', arguments: { command: 'grep -ri "hello" .' } }],
    },
    {
      id: 'a2',
      role: 'assistant',
      rawText: '',
      thinking: '',
      timestamp: '04:41 PM',
      toolCalls: [{ id: 't2', name: 'bash', arguments: { command: 'grep -ri "setting" .' } }],
    },
  ]);

  assert.equal(groups.length, 1);
  assert.equal(groups[0].type, 'message');
  assert.equal(groups[0].msg.role, 'assistant');
  assert.deepEqual(
    groups[0].msg.toolCalls.map((tool) => tool.id),
    ['t1', 't2'],
  );
});

test('does not group assistant tool messages across visible text', () => {
  const groups = computeDisplayGroups([
    {
      id: 'a1',
      role: 'assistant',
      rawText: '',
      thinking: '',
      timestamp: '04:41 PM',
      toolCalls: [{ id: 't1', name: 'bash', arguments: {} }],
    },
    {
      id: 'a2',
      role: 'assistant',
      rawText: 'I found it.',
      thinking: '',
      timestamp: '04:42 PM',
      toolCalls: [],
    },
    {
      id: 'a3',
      role: 'assistant',
      rawText: '',
      thinking: '',
      timestamp: '04:43 PM',
      toolCalls: [{ id: 't2', name: 'bash', arguments: {} }],
    },
  ]);

  assert.equal(groups.length, 3);
  assert.equal(groups[0].msg.toolCalls.length, 1);
  assert.equal(groups[2].msg.toolCalls.length, 1);
});

test('skips assistant messages with no renderable content', () => {
  const groups = computeDisplayGroups([
    {
      id: 'empty',
      role: 'assistant',
      rawText: '',
      thinking: '',
      timestamp: '09:48 AM',
      toolCalls: [],
    },
    {
      id: 'text',
      role: 'assistant',
      rawText: 'Visible response',
      thinking: '',
      timestamp: '09:49 AM',
      toolCalls: [],
    },
  ]);

  assert.equal(groups.length, 1);
  assert.equal(groups[0].msg.id, 'text');
});
