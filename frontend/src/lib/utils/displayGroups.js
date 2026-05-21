function isAssistantToolOnlyMessage(msg) {
  return msg?.role === 'assistant'
    && (msg.toolCalls || []).length > 0
    && !(msg.rawText || '').trim()
    && !(msg.thinking || '').trim();
}

function isEmptyAssistantMessage(msg) {
  return msg?.role === 'assistant'
    && (msg.toolCalls || []).length === 0
    && (msg.images || []).length === 0
    && !(msg.rawText || '').trim()
    && !(msg.thinking || '').trim();
}

function appendAssistantToolMessage(group, msg) {
  group.msg = {
    ...group.msg,
    toolCalls: [
      ...(group.msg.toolCalls || []),
      ...(msg.toolCalls || []),
    ],
  };
}

/**
 * Build the chat render list, grouping adjacent tool-heavy events that have no
 * response text between them.
 */
export function computeDisplayGroups(msgs) {
  const items = [];
  let toolResultGroup = null;
  let assistantToolGroup = null;

  for (const msg of msgs) {
    if (isEmptyAssistantMessage(msg)) {
      continue;
    }

    if (msg.role === 'toolResult') {
      assistantToolGroup = null;
      if (!toolResultGroup) {
        toolResultGroup = { type: 'toolGroup', results: [], groupId: 'tg-' + msg.id };
        items.push(toolResultGroup);
      }
      toolResultGroup.results.push(msg);
      continue;
    }

    toolResultGroup = null;

    if (isAssistantToolOnlyMessage(msg)) {
      if (assistantToolGroup) {
        appendAssistantToolMessage(assistantToolGroup, msg);
      } else {
        assistantToolGroup = {
          type: 'message',
          msg: {
            ...msg,
            toolCalls: [...(msg.toolCalls || [])],
          },
        };
        items.push(assistantToolGroup);
      }
      continue;
    }

    assistantToolGroup = null;
    items.push({ type: 'message', msg });
  }

  return items;
}
