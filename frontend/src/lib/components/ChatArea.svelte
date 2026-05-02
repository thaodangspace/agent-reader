<script>
  import { onMount, tick } from 'svelte';
  import { messages } from '$lib/stores/messages.svelte.js';
  import { rpcRunning, isStreaming, rpcAutoStarting, isRpcRunning } from '$lib/stores/rpc.svelte.js';
  import { activeSession } from '$lib/stores/session.svelte.js';
  import { userScrolledUp, newMessageCount } from '$lib/stores/messages.svelte.js';
  import { sendMessage, abortRPC } from '$lib/actions/rpc.js';
  import { quitSession } from '$lib/actions/session.js';
  import MessageBubble from './MessageBubble.svelte';
  import AssistantBubble from './AssistantBubble.svelte';
  import ToolResultBlock from './ToolResultBlock.svelte';
  import LoadingIndicator from './LoadingIndicator.svelte';
  import ScrollDownButton from './ScrollDownButton.svelte';
  import { isAtBottom, autoResize } from '$lib/utils/scroll.js';

  let input = $state('');
  let textareaEl = $state(null);
  let chatContainer = $state(null);
  let showScrollBtn = $state(false);
  let rpcMap = $state(new Map());

  onMount(() => {
    // Subscribe to rpcRunning store to get reactive updates
    const unsubRpc = rpcRunning.subscribe(map => {
      rpcMap = new Map(map);
    });

    // Subscribe to messages changes for auto-scroll
    const unsubMsgs = messages.subscribe(async msgs => {
      if (!chatContainer || msgs.length === 0) return;

      // Wait for DOM to update with new messages
      await tick();

      // Check if user has scrolled up (tracked by handleScroll on scroll events)
      let scrolledUp = false;
      userScrolledUp.subscribe(v => { scrolledUp = v; })();

      if (!scrolledUp) {
        chatContainer.scrollTop = chatContainer.scrollHeight;
        showScrollBtn = false;
        newMessageCount.set(0);
      } else {
        newMessageCount.update(n => n + 1);
        showScrollBtn = true;
      }
    });

    return () => {
      unsubRpc();
      unsubMsgs();
    };
  });

  // Derive RPC status for the active session
  function activeRpcRunning() {
    return $activeSession ? rpcMap.get($activeSession) === true : false;
  }

  function handleSend() {
    const text = input.trim();
    if (text) {
      sendMessage(text);
      input = '';
      if (textareaEl) autoResize(textareaEl);
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleScroll() {
    if (!chatContainer) return;
    const atBottom = isAtBottom(chatContainer);
    userScrolledUp.set(!atBottom);
    if (atBottom) {
      showScrollBtn = false;
      newMessageCount.set(0);
    } else {
      showScrollBtn = true;
    }
  }

  function scrollToBottomNow() {
    if (chatContainer) {
      chatContainer.scrollTop = chatContainer.scrollHeight;
      userScrolledUp.set(false);
      newMessageCount.set(0);
      showScrollBtn = false;
    }
  }
</script>

<div class="flex-1 flex flex-col min-h-0">
  <!-- Messages -->
  <div
    class="flex-1 overflow-y-auto p-4 flex flex-col gap-3"
    bind:this={chatContainer}
    onscroll={handleScroll}
  >
    {#if $messages.length === 0}
      <div class="flex items-center justify-center h-full text-ctp-overlay0 text-sm">
        {$activeSession ? 'Type a message to start chatting' : 'Select a session to begin'}
      </div>
    {:else}
      {#each $messages as msg (msg.id)}
        {#if msg.role === 'user'}
          <MessageBubble {msg} />
        {:else if msg.role === 'assistant'}
          <AssistantBubble {msg} />
        {:else if msg.role === 'toolResult'}
          <ToolResultBlock {msg} />
        {:else if msg.role === 'system'}
          <div class="flex items-center justify-center animate-fadeIn">
            <div
              class="px-3 py-1.5 rounded-lg text-xs text-ctp-red"
              style="background:color-mix(in srgb, #f38ba8 10%, #1e1e2e)"
            >
              {msg.content}
            </div>
          </div>
        {/if}
      {/each}

      {#if $isStreaming === false && activeRpcRunning()}
        <!-- Show loading indicator while waiting for response -->
      {/if}
    {/if}
  </div>

  <!-- Scroll down button -->
  {#if showScrollBtn}
    <ScrollDownButton onScrollToBottom={scrollToBottomNow} />
  {/if}

  <!-- Input Area -->
  <div class="border-t border-ctp-surface0 bg-ctp-mantle p-3">
    <div class="flex gap-2 items-end">
      <textarea
        bind:this={textareaEl}
        bind:value={input}
        class="flex-1 px-3 py-2 bg-ctp-crust border border-ctp-surface0 rounded-lg text-ctp-text text-sm font-mono resize-none focus:outline-none focus:border-ctp-blue placeholder:text-ctp-overlay0"
        rows="1"
        placeholder={$activeSession ? 'Type a message...' : 'Select a session to begin...'}
        disabled={$activeSession === null}
        onkeydown={handleKeydown}
        oninput={() => autoResize(textareaEl)}
      ></textarea>
      <button
        class="px-4 py-2 rounded-lg text-sm font-semibold bg-ctp-blue text-ctp-crust hover:bg-ctp-blue/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
        disabled={$activeSession === null}
        onclick={handleSend}
      >
        {$isStreaming ? 'Queue' : ($rpcAutoStarting ? 'Starting...' : 'Send')}
      </button>
    </div>

    <!-- Status Bar -->
    <div class="text-[10px] text-ctp-overlay0 mt-1.5 flex items-center gap-3 justify-between">
      <div class="flex items-center gap-3">
        <span>Enter to send · Shift+Enter for new line</span>
        {#if $isStreaming}
          <span>Streaming... messages will be queued</span>
        {/if}
        {#if $rpcAutoStarting}
          <span class="text-ctp-yellow">Starting RPC...</span>
        {/if}
      </div>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <div
            class="w-2 h-2 rounded-full transition-colors duration-300"
            style="background: {activeRpcRunning() || $rpcAutoStarting ? '#a6e3a1' : '#6c7086'}"
          ></div>
          <span>{$rpcAutoStarting ? 'RPC: starting' : (activeRpcRunning() ? 'RPC: active' : 'RPC: idle')}</span>
        </div>
        {#if $isStreaming}
          <button
            class="px-3 py-1 rounded-md text-xs font-semibold bg-ctp-red/20 text-ctp-red hover:bg-ctp-red/30 transition-colors"
            onclick={abortRPC}
          >
            ⏹ Abort
          </button>
        {/if}
        {#if $activeSession && activeRpcRunning()}
          <button
            class="px-3 py-1 rounded-md text-xs font-semibold bg-ctp-red/20 text-ctp-red hover:bg-ctp-red/30 transition-colors"
            onclick={quitSession}
          >
            Quit Session
          </button>
        {/if}
      </div>
    </div>
  </div>
</div>
