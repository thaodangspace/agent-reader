// ===== Chat Rendering Module =====
import {
  chatMessages, activeSession, currentAssistantEl,
  setIsStreaming, setCurrentAssistantEl,
  userScrolledUp, newMessageCount, setUserScrolledUp, setNewMessageCount
} from './state.js';
import { scrollToBottom, isAtBottom, escapeHTML, formatText, timeNow } from './utils.js';
import { updateStreamingUI } from './ui.js';

// ===== Loading Indicator =====
let loadingIndicator = null;

export function showLoadingIndicator() {
  clearEmptyState();
  if (loadingIndicator) return;

  loadingIndicator = document.createElement('div');
  loadingIndicator.id = 'loadingIndicator';
  loadingIndicator.className = 'flex flex-col items-start animate-fadeIn';
  loadingIndicator.innerHTML = `
    <div class="px-4 py-2.5 rounded-2xl text-sm" style="background:color-mix(in srgb, #a6e3a1 20%, #313244);">
      <div class="flex items-center gap-2">
        <div class="typing-dots flex gap-1">
          <span class="w-2 h-2 rounded-full bg-ctp-green" style="animation: pulse 1.4s ease-in-out infinite;"></span>
          <span class="w-2 h-2 rounded-full bg-ctp-green" style="animation: pulse 1.4s ease-in-out infinite 0.2s;"></span>
          <span class="w-2 h-2 rounded-full bg-ctp-green" style="animation: pulse 1.4s ease-in-out infinite 0.4s;"></span>
        </div>
        <span class="text-xs text-ctp-overlay0">Waiting for response...</span>
      </div>
    </div>
  `;
  chatMessages.appendChild(loadingIndicator);
  scrollToBottom(chatMessages, userScrolledUp);
}

export function hideLoadingIndicator() {
  if (loadingIndicator) {
    loadingIndicator.remove();
    loadingIndicator = null;
  }
}

// ===== Scroll Tracking =====
let scrollTrackingInitialized = false;

function initScrollTracking() {
  if (scrollTrackingInitialized) return;
  scrollTrackingInitialized = true;

  chatMessages.addEventListener('scroll', () => {
    const atBottom = isAtBottom(chatMessages);
    const wasUp = userScrolledUp;

    setUserScrolledUp(!atBottom);

    if (atBottom && wasUp) {
      // User scrolled back to bottom - hide button, reset count
      hideScrollDownButton();
    } else if (!atBottom && isStreaming) {
      // User scrolled up while streaming - show button
      showScrollDownButton();
    }
  });
}

// Check scroll position without relying on event — call before each scroll
function checkScrollPosition() {
  const atBottom = isAtBottom(chatMessages);
  setUserScrolledUp(!atBottom);
}

function showScrollDownButton() {
  let btn = document.getElementById('scrollDownBtn');
  if (!btn) {
    btn = document.createElement('button');
    btn.id = 'scrollDownBtn';
    btn.className = 'fixed bottom-24 right-6 z-40 flex items-center gap-2 px-4 py-2.5 rounded-full text-xs font-semibold shadow-lg transition-all animate-fadeIn';
    btn.style.background = 'color-mix(in srgb, #89b4fa 25%, #313244)';
    btn.style.color = '#cdd6f4';
    btn.style.border = '1px solid #89b4fa';
    btn.onclick = scrollToBottomNow;
    document.body.appendChild(btn);
  }
  const count = newMessageCount;
  btn.innerHTML = count > 0
    ? `<span>↓</span><span>New ${count} message${count > 1 ? 's' : ''}</span>`
    : `<span>↓</span><span>Scroll to bottom</span>`;
  btn.classList.remove('hidden');
}

function hideScrollDownButton() {
  const btn = document.getElementById('scrollDownBtn');
  if (btn) {
    btn.classList.add('hidden');
  }
  setNewMessageCount(0);
}

function scrollToBottomNow() {
  chatMessages.scrollTop = chatMessages.scrollHeight;
  setUserScrolledUp(false);
  setNewMessageCount(0);
  hideScrollDownButton();
}

// ===== Deduplication =====
const seenEvents = new Set();

export function clearSeenEvents() {
  seenEvents.clear();
}

// ===== WS Message Handler =====
export function onWSMessage(msg) {
  const data = msg.data;
  if (!data || !data.type) return;

  if (data.type === 'session') {
    htmx.ajax('GET', '/sessions', { target: '#sessionList', swap: 'innerHTML' });
    return;
  }

  if (activeSession && msg.session_id !== activeSession) return;

  // Deduplicate by data.id (replay + watcher can send same events)
  if (data.id && seenEvents.has(data.id)) return;
  if (data.id) seenEvents.add(data.id);

  switch (data.type) {
    case 'message':
      renderLegacyMessage(data);
      break;

    case 'agent_start':
      setIsStreaming(true);
      updateStreamingUI(true);
      initScrollTracking();
      hideLoadingIndicator();
      break;

    case 'agent_end':
      setIsStreaming(false);
      updateStreamingUI(false);
      setCurrentAssistantEl(null);
      if (isAtBottom(chatMessages)) {
        scrollToBottom(chatMessages, false, true);
        hideScrollDownButton();
      }
      break;

    case 'turn_start':
      setIsStreaming(true);
      updateStreamingUI(true);
      initScrollTracking();
      hideLoadingIndicator();
      break;

    case 'message_update': {
      const ev = data.assistantMessageEvent;
      if (!ev) break;
      if (ev.type === 'text_delta') {
        appendAssistantText(ev.delta);
      } else if (ev.type === 'thinking_delta') {
        appendAssistantThinking(ev.delta);
      } else if (ev.type === 'toolcall_start') {
        startToolCall(ev);
      } else if (ev.type === 'toolcall_end') {
        endToolCall(ev);
      } else if (ev.type === 'done') {
        setIsStreaming(false);
        updateStreamingUI(false);
      }
      break;
    }

    case 'message_end': {
      const msg = data.message;
      if (msg && msg.role === 'user') {
        hideLoadingIndicator();
        addUserMessage(msg.content);
      } else if (msg && msg.role === 'toolResult') {
        addToolResult(msg);
      }
      break;
    }
  }
}

// ===== Chat Rendering Helpers =====
function clearEmptyState() {
  const es = document.getElementById('emptyState');
  if (es && es.parentNode) es.remove();
}

export function addUserMessage(content) {
  clearEmptyState();
  const text = typeof content === 'string' ? content :
               Array.isArray(content) ? content.filter(c => c.type === 'text').map(c => typeof c.text === 'string' ? c.text : String(c.text ?? '')).join('') : '';
  if (!text) return;

  const row = document.createElement('div');
  row.className = 'flex flex-col items-end animate-fadeIn';
  row.innerHTML = `
    <div class="px-4 py-2.5 rounded-2xl text-sm leading-relaxed break-words max-w-[75%] message-bubble"
         style="background:color-mix(in srgb, #89b4fa 25%, #313244); border-bottom-right-radius: 4px;">
      <div class="prose-markdown">${formatText(text)}</div>
    </div>
    <div class="text-[10px] text-ctp-overlay0 mt-0.5 px-1 text-right">${timeNow()}</div>
  `;
  chatMessages.appendChild(row);
  scrollToBottom(chatMessages, userScrolledUp);
}

function startAssistantMessage() {
  clearEmptyState();
  const row = document.createElement('div');
  row.className = 'flex flex-col items-start animate-fadeIn';
  row.innerHTML = `
    <div class="px-4 py-2.5 rounded-2xl text-sm leading-relaxed break-words max-w-[75%] message-bubble assistant-bubble"
         style="background:color-mix(in srgb, #a6e3a1 20%, #313244); border-bottom-left-radius: 4px;">
      <div class="assistant-content prose-markdown" data-raw=""></div>
    </div>
    <div class="text-[10px] text-ctp-overlay0 mt-0.5 px-1">${timeNow()}</div>
  `;
  chatMessages.appendChild(row);
  setCurrentAssistantEl(row);
  checkScrollPosition();
  if (!userScrolledUp) {
    scrollToBottom(chatMessages, false);
  }
  return row.querySelector('.assistant-content');
}

function appendAssistantText(delta) {
  if (typeof delta !== 'string') {
    delta = delta == null ? '' : String(delta);
  }
  let content = currentAssistantEl ? currentAssistantEl.querySelector('.assistant-content') : null;
  if (!content) {
    content = startAssistantMessage();
  }
  content.dataset.raw = (content.dataset.raw || '') + delta;
  content.innerHTML = formatText(content.dataset.raw);
  checkScrollPosition();
  if (!userScrolledUp) {
    scrollToBottom(chatMessages, false);
  }
  if (userScrolledUp) {
    setNewMessageCount(newMessageCount + 1);
    showScrollDownButton();
  }
}

function appendAssistantThinking(delta) {
  if (typeof delta !== 'string') {
    delta = delta == null ? '' : String(delta);
  }
  let content = currentAssistantEl ? currentAssistantEl.querySelector('.assistant-content') : null;
  if (!content) content = startAssistantMessage();

  let thinkingEl = content.querySelector('.thinking-block');
  if (!thinkingEl) {
    thinkingEl = document.createElement('div');
    thinkingEl.className = 'thinking-block rounded-lg overflow-hidden border border-ctp-surface0 mb-2';
    thinkingEl.style.background = 'color-mix(in srgb, #cba6f7 8%, #313244)';
    thinkingEl.innerHTML = `
      <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer" onclick="toggleCollapse(this)">
        <span class="transition-transform duration-200 text-[10px]">▶</span>
        <span>💭</span>
        <span class="font-semibold text-ctp-mauve">Thinking</span>
      </button>
      <div class="thinking-body hidden border-t border-ctp-surface0">
        <div class="p-3 text-xs" style="background:color-mix(in srgb, #1e1e2e 50%, #11111b);">
          <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto text-ctp-mauve opacity-80"></pre>
        </div>
      </div>
    `;
    content.insertBefore(thinkingEl, content.firstChild);
  }
  thinkingEl.querySelector('pre').textContent += delta;
  checkScrollPosition();
  if (!userScrolledUp) {
    scrollToBottom(chatMessages, false);
  }
  if (userScrolledUp) {
    setNewMessageCount(newMessageCount + 1);
    showScrollDownButton();
  }
}

function startToolCall(ev) {
  let content = currentAssistantEl ? currentAssistantEl.querySelector('.assistant-content') : null;
  if (!content) content = startAssistantMessage();
  const toolName = ev.toolCall?.name || 'unknown';
  const toolId = ev.toolCall?.id || '';
  const toolEl = document.createElement('div');
  toolEl.className = 'tool-call-block rounded-lg overflow-hidden border border-ctp-surface0 mb-2';
  toolEl.style.background = 'color-mix(in srgb, #fab387 10%, #313244)';
  toolEl.dataset.toolId = toolId;
  toolEl.innerHTML = `
    <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer" onclick="toggleCollapse(this)">
      <span class="transition-transform duration-200 text-[10px]">▶</span>
      <span>🔧</span>
      <span class="font-semibold" style="color:#fab387">${escapeHTML(toolName)}</span>
      <span class="text-ctp-overlay0 text-[10px] ml-auto tool-args-preview"></span>
    </button>
    <div class="thinking-body hidden border-t border-ctp-surface0">
      <div class="p-3 text-xs overflow-x-auto" style="background:color-mix(in srgb, #1e1e2e 50%, #11111b);">
        <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto tool-args-text"></pre>
      </div>
    </div>
  `;
  content.appendChild(toolEl);
  checkScrollPosition();
  if (!userScrolledUp) {
    scrollToBottom(chatMessages, false);
  }
  if (userScrolledUp) {
    setNewMessageCount(newMessageCount + 1);
    showScrollDownButton();
  }
}

function endToolCall(ev) {
  if (!ev.toolCall) return;
  const toolId = ev.toolCall.id || '';
  const toolEl = currentAssistantEl?.querySelector(`[data-tool-id="${toolId}"]`);
  if (!toolEl) return;
  const args = ev.toolCall.arguments || {};
  const argsStr = typeof args === 'string' ? args : JSON.stringify(args, null, 2);
  const argsText = toolEl.querySelector('.tool-args-text');
  const argsPreview = toolEl.querySelector('.tool-args-preview');
  if (argsText) argsText.textContent = argsStr.length > 200 ? argsStr.substring(0, 200) + '...' : argsStr;
  if (argsPreview) argsPreview.textContent = argsStr.length > 50 ? argsStr.substring(0, 50) + '...' : argsStr;
}

export function addToolResult(msg) {
  clearEmptyState();
  const isError = msg.isError || false;
  const toolName = msg.toolName || 'unknown';
  let content = '';
  if (msg.content) {
    if (typeof msg.content === 'string') content = msg.content;
    else if (Array.isArray(msg.content)) content = msg.content.filter(c => c.type === 'text').map(c => typeof c.text === 'string' ? c.text : String(c.text ?? '')).join('');
  }

  const row = document.createElement('div');
  row.className = 'flex flex-col items-start animate-fadeIn w-full';
  const borderColor = isError ? '#f38ba8' : '#585b70';
  const headerBg = isError ? 'color-mix(in srgb, #f38ba8 15%, #313244)' : 'color-mix(in srgb, #f9e2af 15%, #313244)';

  row.innerHTML = `
    <div class="w-full max-w-[85%] rounded-xl overflow-hidden border border-ctp-surface0" style="border-color:${borderColor}">
      <button class="w-full flex items-center gap-2 px-3 py-2 text-xs cursor-pointer" style="background:${headerBg}" onclick="toggleCollapse(this)">
        <span class="transition-transform duration-200 text-[10px]">▶</span>
        <span>📎</span>
        <span class="font-semibold ${isError ? 'text-ctp-red' : 'text-ctp-yellow'}">${escapeHTML(toolName)}</span>
        ${isError ? '<span class="text-ctp-red text-[10px] ml-auto">Error</span>' : '<span class="text-ctp-overlay0 text-[10px] ml-auto">Result</span>'}
      </button>
      <div class="thinking-body hidden border-t border-ctp-surface0">
        <div class="p-3 text-xs overflow-x-auto" style="background:color-mix(in srgb, #1e1e2e 50%, #11111b);">
          <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[400px] overflow-y-auto">${escapeHTML(content || '(no output)')}</pre>
        </div>
      </div>
    </div>
  `;
  chatMessages.appendChild(row);
  scrollToBottom(chatMessages, userScrolledUp);
}

// ===== Legacy: render file-watcher message events =====
function renderLegacyMessage(data) {
  const msg = data.message || {};
  const role = msg.role || 'unknown';
  if (role !== 'user' && role !== 'assistant' && role !== 'toolResult') return;

  clearEmptyState();
  const row = document.createElement('div');
  row.className = `flex flex-col w-full animate-fadeIn ${role === 'user' ? 'items-end' : 'items-start'}`;
  const time = data.timestamp ? new Date(data.timestamp).toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'}) : '';

  let bubbleContent = '';
  if (msg.content && msg.content.length > 0) {
    msg.content.forEach((block) => {
      if (block.type === 'text') {
        bubbleContent += `<div class="prose-markdown">${formatText(block.text || '')}</div>`;
      } else if (block.type === 'thinking') {
        const t = block.thinking || '';
        bubbleContent += `
          <div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2" style="background:color-mix(in srgb, #cba6f7 8%, #313244)">
            <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer" onclick="toggleCollapse(this)">
              <span class="transition-transform duration-200 text-[10px]">▶</span>
              <span>💭</span>
              <span class="font-semibold text-ctp-mauve">Thinking</span>
              <span class="text-ctp-overlay0 text-[10px] ml-auto">${escapeHTML(t.substring(0, 60))}…</span>
            </button>
            <div class="thinking-body hidden border-t border-ctp-surface0">
              <div class="p-3 text-xs" style="background:color-mix(in srgb, #1e1e2e 50%, #11111b);">
                <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto text-ctp-mauve opacity-80">${escapeHTML(t)}</pre>
              </div>
            </div>
          </div>`;
      } else if (block.type === 'toolCall') {
        const name = block.toolCallName || block.name || 'unknown';
        let argsStr = '';
        if (block.arguments) argsStr = typeof block.arguments === 'string' ? block.arguments : JSON.stringify(block.arguments, null, 2);
        bubbleContent += `
          <div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2" style="background:color-mix(in srgb, #fab387 10%, #313244)">
            <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer" onclick="toggleCollapse(this)">
              <span class="transition-transform duration-200 text-[10px]">▶</span>
              <span>🔧</span>
              <span class="font-semibold" style="color:#fab387">${escapeHTML(name)}</span>
              <span class="text-ctp-overlay0 text-[10px] ml-auto">${escapeHTML(argsStr.substring(0, 50))}…</span>
            </button>
            <div class="thinking-body hidden border-t border-ctp-surface0">
              <div class="p-3 text-xs" style="background:color-mix(in srgb, #1e1e2e 50%, #11111b);">
                <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto">${escapeHTML(argsStr)}</pre>
              </div>
            </div>
          </div>`;
      }
    });
  }

  if (!bubbleContent) bubbleContent = '<span class="opacity-50">(empty)</span>';

  const bg = role === 'user'
    ? 'background:color-mix(in srgb, #89b4fa 25%, #313244); border-bottom-right-radius: 4px;'
    : role === 'toolResult'
      ? 'background:color-mix(in srgb, #f9e2af 15%, #313244);'
      : 'background:color-mix(in srgb, #a6e3a1 20%, #313244); border-bottom-left-radius: 4px;';

  row.innerHTML = `
    <div class="px-3.5 py-2.5 rounded-2xl text-sm leading-relaxed break-words max-w-[75%] message-bubble" style="${bg}">${bubbleContent}</div>
    <div class="text-[10px] text-ctp-overlay0 mt-0.5 px-1 ${role === 'user' ? 'text-right' : ''}">${time}</div>
  `;
  chatMessages.appendChild(row);
  scrollToBottom(chatMessages, userScrolledUp);
}

// ===== System Messages =====
export function addSystemMessage(text) {
  clearEmptyState();
  const row = document.createElement('div');
  row.className = 'flex flex-col items-center animate-fadeIn';
  row.innerHTML = `
    <div class="px-3 py-1.5 rounded-lg text-xs text-ctp-red" style="background:color-mix(in srgb, #f38ba8 10%, #313244)">
      ${escapeHTML(text)}
    </div>
  `;
  chatMessages.appendChild(row);
  scrollToBottom(chatMessages, userScrolledUp);
}
