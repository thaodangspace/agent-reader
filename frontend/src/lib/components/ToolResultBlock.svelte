<script>
  import { escapeHTML, highlightCode } from '$lib/utils/markdown.js';
  import { detectLanguageFromPath } from '$lib/utils/language.js';
  import { unescapeJsonString } from '$lib/utils/json.js';

  let { msg, standalone = true } = $props();
  let collapsed = $state(true); 

  $effect(() => {
    // Reset collapsed state when msg changes
    collapsed = msg.toolName !== 'bash';
  });

  // Derive content and highlighted HTML from msg prop
  let content = $derived.by(() => {
    return unescapeJsonString(msg.content || '(no output)');
  });

  let contentHTML = $derived.by(() => {
    const c = content;
    if (msg.toolName === 'read') {
      const lang = msg.language || detectLanguageFromPath(msg.filePath || '');
      if (lang) {
        return highlightCode(c, lang);
      }
    }
    return escapeHTML(c);
  });

  let isError = $derived(msg.isError || false);
  let highlighted = $derived(msg.toolName === 'read' && (msg.language || detectLanguageFromPath(msg.filePath || '')));

  function parseAnswers(resultStr) {
    if (!resultStr) return [];
    const answers = [];
    const regex = /"([^"]+)"="([^"]+)"/g;
    let match;
    try {
      while ((match = regex.exec(resultStr)) !== null) {
        answers.push({
          question: match[1],
          answer: match[2]
        });
      }
    } catch {}
    return answers;
  }

  function toggle() {
    collapsed = !collapsed;
  }
</script>

{#if standalone}
  <div class="flex flex-col items-start animate-fadeIn w-full">
    <div
      class="w-full max-w-[85%] rounded-xl overflow-hidden border border-ctp-surface0"
      style="border-color: {isError ? '#e95f59' : '#e5e5e5'}"
    >
      {@render toolContent()}
    </div>
  </div>
{:else}
  <div class="rounded-xl overflow-hidden" style="border-color: {isError ? '#e95f59' : '#e5e5e5'}">
    {@render toolContent()}
  </div>
{/if}

{#snippet toolContent()}
  <button
    class="w-full flex items-center gap-2 px-3 py-2 text-xs cursor-pointer text-left"
    style="background: {isError
      ? 'color-mix(in srgb, #e95f59 12%, #ffffff)'
      : 'color-mix(in srgb, #dbab09 12%, #ffffff)'}"
    onclick={toggle}
  >
    <span
      class="transition-transform duration-200 text-[10px]"
      style="transform: {collapsed ? '' : 'rotate(90deg)'}"
    >▶</span>
    <span>📎</span>
    <span class="font-semibold {isError ? 'text-ctp-red' : 'text-ctp-yellow'}">{escapeHTML(msg.toolName)}</span>
    {#if isError}
      <span class="text-ctp-red text-[10px] ml-auto">Error</span>
    {:else}
      <span class="text-ctp-overlay0 text-[10px] ml-auto">Result</span>
    {/if}
  </button>
  <div class="border-t border-ctp-surface0" class:hidden={collapsed}>
    {#if msg.toolName?.toLowerCase() === 'askuserquestion'}
      {@const parsed = parseAnswers(content)}
      <div class="p-4 bg-ctp-base text-ctp-text text-left">
        <div class="font-bold text-ctp-text mb-3 flex items-center gap-1.5 pb-2 border-b border-ctp-crust">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-ctp-green shrink-0" viewBox="0 0 24 24" fill="none" 
               stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
          </svg>
          <span class="text-[11px] uppercase tracking-wider font-bold">Answers Submitted</span>
        </div>
        {#if parsed.length > 0}
          <div class="flex flex-col gap-3">
            {#each parsed as item}
              <div class="flex flex-col gap-1 text-xs">
                <span class="font-semibold text-ctp-text">{item.question}</span>
                <span class="text-ctp-green font-bold pl-4 flex items-center gap-1.5" style="color: var(--color-ctp-green)">
                  <span class="w-1.5 h-1.5 rounded-full bg-ctp-green" style="background: var(--color-ctp-green)"></span>
                  {item.answer}
                </span>
              </div>
            {/each}
          </div>
        {:else}
          <div class="italic text-ctp-overlay1 font-mono whitespace-pre-wrap break-words pl-5.5 text-[11px]">
            {content}
          </div>
        {/if}
      </div>
    {:else}
      <div class="p-3 text-xs overflow-x-auto" style="background:#f6f6f6;">
        {#if highlighted}
          <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[400px] overflow-y-auto">
            {@html contentHTML}
          </pre>
        {:else}
          <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[400px] overflow-y-auto">{contentHTML}</pre>
        {/if}
      </div>
    {/if}
  </div>
{/snippet}
