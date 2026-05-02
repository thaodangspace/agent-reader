import { messages } from '$lib/stores/messages.svelte.js';
import { activeSession, sessions } from '$lib/stores/session.svelte.js';
import { isStreaming } from '$lib/stores/rpc.svelte.js';
import { fetchSessions } from '$lib/api/sessions.js';
import { detectLanguageFromPath } from '$lib/utils/language.js';
import { unescapeJsonString } from '$lib/utils/json.js';

// Deduplication
const seenEvents = new Set();

export function clearSeenEvents() {
  seenEvents.clear();
}

// Tool Call Language Tracking
const toolCallLanguages = new Map();

export function onWSMessage(msg) {
  const data = msg.data;
  if (!data || !data.type) return;

  // Session list update
  if (data.type === 'session') {
    refreshSessions();
    return;
  }

  // Filter by active session
  let currentSession = null;
  const unsub = activeSession.subscribe(id => { currentSession = id; });
  unsub();
  if (currentSession && msg.session_id !== currentSession) return;

  // Deduplicate
  if (data.id && seenEvents.has(data.id)) return;
  if (data.id) seenEvents.add(data.id);

  switch (data.type) {
    case 'agent_start':
    case 'turn_start':
      isStreaming.set(true);
      break;

    case 'agent_end':
      isStreaming.set(false);
      break;

    case 'message_update': {
      const ev = data.assistantMessageEvent;
      if (!ev) break;
      appendToCurrentAssistant(ev);
      break;
    }

    case 'message_end': {
      const msg = data.message;
      if (msg?.role === 'user') {
        addUserMessage(msg.content);
      } else if (msg?.role === 'toolResult') {
        addToolResult(msg);
      }
      break;
    }

    case 'message':
      renderLegacyMessage(data);
      break;
  }
}

function addUserMessage(content) {
  const text = typeof content === 'string' ? content : extractText(content);
  if (!text) return;

  messages.update(msgs => [...msgs, {
    id: crypto.randomUUID(),
    role: 'user',
    content: text,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
  }]);
}

let currentAssistantId = null;

function appendToCurrentAssistant(ev) {
  if (ev.type === 'text_delta') {
    if (!currentAssistantId) {
      const id = crypto.randomUUID();
      currentAssistantId = id;
      messages.update(msgs => [...msgs, {
        id,
        role: 'assistant',
        rawText: ev.delta,
        thinking: '',
        toolCalls: [],
        isStreaming: true,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      }]);
    } else {
      messages.update(msgs => msgs.map(m => {
        if (m.id === currentAssistantId) {
          return { ...m, rawText: (m.rawText || '') + ev.delta, isStreaming: true };
        }
        return m;
      }));
    }
  } else if (ev.type === 'thinking_delta') {
    messages.update(msgs => msgs.map(m => {
      if (m.id === currentAssistantId) {
        return { ...m, thinking: (m.thinking || '') + ev.delta };
      }
      return m;
    }));
  } else if (ev.type === 'toolcall_start') {
    messages.update(msgs => msgs.map(m => {
      if (m.id === currentAssistantId) {
        return {
          ...m,
          toolCalls: [...(m.toolCalls || []), {
            id: ev.toolCall?.id || '',
            name: ev.toolCall?.name || 'unknown',
            arguments: {},
          }],
        };
      }
      return m;
    }));
  } else if (ev.type === 'toolcall_end') {
    messages.update(msgs => msgs.map(m => {
      if (m.id === currentAssistantId) {
        const updatedCalls = (m.toolCalls || []).map(tc =>
          tc.id === ev.toolCall?.id
            ? { ...tc, arguments: ev.toolCall?.arguments || {} }
            : tc
        );
        // Detect language for read tool
        if (ev.toolCall?.name === 'read') {
          const tc = ev.toolCall;
          const args = tc.arguments || {};
          const argsStr = typeof args === 'string' ? args : JSON.stringify(args, null, 2);
          if (argsStr) {
            try {
              const parsed = typeof args === 'string' ? JSON.parse(args) : args;
              if (parsed?.path) {
                const lang = detectLanguageFromPath(parsed.path);
                if (lang) toolCallLanguages.set(tc.id, lang);
              }
            } catch {}
          }
        }
        return { ...m, toolCalls: updatedCalls };
      }
      return m;
    }));
  } else if (ev.type === 'done') {
    isStreaming.set(false);
    currentAssistantId = null;
  }
}

function addToolResult(msg) {
  // Look up language from stored toolCall info
  const toolCallId = msg.toolCallId || '';
  const lang = toolCallLanguages.get(toolCallId);
  const toolName = msg.toolName || 'unknown';

  let content = '';
  if (msg.content) {
    if (typeof msg.content === 'string') content = msg.content;
    else if (Array.isArray(msg.content)) content = extractText(msg.content);
  }

  content = unescapeJsonString(content);

  const filePath = extractFilePath(msg);

  messages.update(msgs => [...msgs, {
    id: crypto.randomUUID(),
    role: 'toolResult',
    toolName: toolName,
    content: content || '(no output)',
    isError: msg.isError || false,
    toolCallId: toolCallId,
    filePath: filePath,
    language: (toolName === 'read' && lang) ? lang : null,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
  }]);
}

function renderLegacyMessage(data) {
  const msg = data.message || {};
  const role = msg.role || 'unknown';
  if (role !== 'user' && role !== 'assistant' && role !== 'toolResult') return;

  const time = data.timestamp ? new Date(data.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '';

  if (role === 'user') {
    const content = msg.content || [];
    const text = Array.isArray(content)
      ? content.filter(c => c.type === 'text').map(c => typeof c.text === 'string' ? c.text : String(c.text ?? '')).join('')
      : String(content);
    if (text) {
      messages.update(msgs => [...msgs, {
        id: crypto.randomUUID(),
        role: 'user',
        content: text,
        timestamp: time,
      }]);
    }
  } else if (role === 'assistant') {
    const content = msg.content || [];
    let rawText = '';
    let thinking = '';
    const toolCalls = [];

    content.forEach(block => {
      if (block.type === 'text') {
        rawText += typeof block.text === 'string' ? block.text : String(block.text ?? '');
      } else if (block.type === 'thinking') {
        thinking += block.thinking || '';
      } else if (block.type === 'toolCall') {
        const name = block.toolCallName || block.name || 'unknown';
        const args = block.arguments || {};
        const toolId = block.id || '';
        toolCalls.push({ id: toolId, name, arguments: args });
      }
    });

    messages.update(msgs => [...msgs, {
      id: crypto.randomUUID(),
      role: 'assistant',
      rawText,
      thinking,
      toolCalls,
      isStreaming: false,
      timestamp: time,
    }]);
  } else if (role === 'toolResult') {
    let content = '';
    if (msg.content) {
      if (typeof msg.content === 'string') content = msg.content;
      else if (Array.isArray(msg.content)) content = extractText(msg.content);
    }
    content = unescapeJsonString(content);

    messages.update(msgs => [...msgs, {
      id: crypto.randomUUID(),
      role: 'toolResult',
      toolName: msg.toolName || 'unknown',
      content: content || '(no output)',
      isError: msg.isError || false,
      toolCallId: msg.toolCallId || '',
      filePath: extractFilePath(msg),
      language: null,
      timestamp: time,
    }]);
  }
}

export function addSystemMessage(text) {
  messages.update(msgs => [...msgs, {
    id: crypto.randomUUID(),
    role: 'system',
    content: text,
  }]);
}

function extractText(content) {
  if (typeof content === 'string') return content;
  if (Array.isArray(content)) {
    return content
      .filter(c => c.type === 'text')
      .map(c => typeof c.text === 'string' ? c.text : String(c.text ?? ''))
      .join('');
  }
  return '';
}

function extractFilePath(msg) {
  if (msg.toolName === 'read' && msg.content) {
    return msg.filePath || '';
  }
  return '';
}

async function refreshSessions() {
  try {
    const list = await fetchSessions();
    sessions.set(list);
  } catch (e) {
    console.error('Failed to refresh sessions:', e);
  }
}

export { toolCallLanguages };
