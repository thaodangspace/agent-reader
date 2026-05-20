<script>
  import { escapeHTML, highlightCode } from '$lib/utils/markdown.js';
  import { detectLanguageFromPath } from '$lib/utils/language.js';
  import { unescapeJsonString } from '$lib/utils/json.js';
  import DiffView from './DiffView.svelte';
  import { ChevronRight, ChevronDown, FileText, BookOpen, Terminal, Wrench } from '@lucide/svelte';

  let { tc } = $props();
  let collapsed = $state(true);

  // Parse arguments for structured display
  let argsStr = $state(
    typeof tc.arguments === 'string' ? tc.arguments : JSON.stringify(tc.arguments || {}, null, 2)
  );

  let parsedArgs = $derived.by(() => {
    try {
      return typeof tc.arguments === 'string' ? JSON.parse(tc.arguments) : tc.arguments;
    } catch {
      return null;
    }
  });

  let isEditTool = $derived(tc.name === 'edit' && parsedArgs);
  let isWriteTool = $derived(tc.name === 'write' && parsedArgs);
  let isReadTool = $derived(tc.name === 'read');

  let hasResult = $derived(tc.result !== undefined && tc.result !== null);
  let resultContent = $derived(hasResult ? unescapeJsonString(tc.result || '(no output)') : '');

  let writePath = $derived(parsedArgs?.path || '');
  let writeLang = $derived(detectLanguageFromPath(writePath) || '');
  let writeContent = $derived(parsedArgs?.content || '');
  let writeContentHTML = $derived(
    writeLang ? highlightCode(writeContent, writeLang) : escapeHTML(writeContent)
  );

  // Result highlighting
  let resultHTML = $derived.by(() => {
    if (!hasResult) return '';
    if (tc.resultLanguage) return highlightCode(resultContent, tc.resultLanguage);
    if (tc.name === 'read' && tc.resultFilePath) {
      const lang = detectLanguageFromPath(tc.resultFilePath);
      if (lang) return highlightCode(resultContent, lang);
    }
    return escapeHTML(resultContent);
  });

  let resultIsError = $derived(tc.resultIsError || false);

  function toggle() {
    collapsed = !collapsed;
  }
</script>

{#if isEditTool}
  <DiffView filePath={parsedArgs.path} edits={parsedArgs.edits} />
{:else if isWriteTool}
  <div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2" style="background:color-mix(in srgb, #135ce0 8%, #f6f6f6)">
    <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer" onclick={toggle}>
      <span class="flex items-center">
        {#if collapsed}
          <ChevronRight size={12} />
        {:else}
          <ChevronDown size={12} />
        {/if}
      </span>
      <FileText size={14} class="text-ctp-blue" />
      <span class="font-semibold" style="color:#135ce0">write</span>
      <span class="text-ctp-overlay0 text-[10px] ml-auto truncate max-w-[300px]" title={writePath}>{writePath.split('/').slice(-2).join('/')}</span>
    </button>
    <div class="border-t border-ctp-surface0" class:hidden={collapsed}>
      <div class="text-[11px] font-mono" style="background:color-mix(in srgb, #ffffff 50%, #f6f6f6);">
        <pre class="p-3 overflow-x-auto max-h-[400px] overflow-y-auto whitespace-pre-wrap break-words">
          {@html writeContentHTML}
        </pre>
      </div>
    </div>
  </div>
{:else}
  <div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2"
       style="background: {hasResult && resultIsError ? 'color-mix(in srgb, #e95f59 8%, #ffffff)' : 'color-mix(in srgb, #135ce0 6%, #ffffff)'}">
    <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer" onclick={toggle}>
      <span class="flex items-center">
        {#if collapsed}
          <ChevronRight size={12} />
        {:else}
          <ChevronDown size={12} />
        {/if}
      </span>
      {#if tc.name === 'read'}
        <BookOpen size={14} class="text-ctp-blue" />
      {:else if tc.name === 'bash'}
        <Terminal size={14} class="text-ctp-blue" />
      {:else}
        <Wrench size={14} class="text-ctp-blue" />
      {/if}
      <span class="font-semibold" style="color:#135ce0">{escapeHTML(tc.name)}</span>
      {#if parsedArgs?.path}
        <span class="text-ctp-overlay0 text-[10px] ml-auto truncate max-w-[300px]" title={parsedArgs.path}>{parsedArgs.path.split('/').slice(-2).join('/')}</span>
      {:else}
        <span class="text-ctp-overlay0 text-[10px] ml-auto">{escapeHTML(argsStr.substring(0, 50))}…</span>
      {/if}
    </button>
    <div class="border-t border-ctp-surface0" class:hidden={collapsed}>
      <!-- Arguments section -->
      {#if tc.name !== 'read'}
        <div class="p-3 text-xs overflow-x-auto" style="background:color-mix(in srgb, #ffffff 50%, #f6f6f6);">
          <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto">{argsStr}</pre>
        </div>
      {:else}
        <!-- For read: show content directly (result = file content) -->
        {#if hasResult}
          <div class="p-0 text-[11px] font-mono" style="background:color-mix(in srgb, #ffffff 50%, #f6f6f6);">
            <pre class="p-3 overflow-x-auto max-h-[400px] overflow-y-auto whitespace-pre-wrap break-words">
              {@html resultHTML}
            </pre>
          </div>
        {:else}
          <div class="p-3 text-xs overflow-x-auto" style="background:color-mix(in srgb, #ffffff 50%, #f6f6f6);">
            <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto">{argsStr}</pre>
          </div>
        {/if}
      {/if}
      <!-- Result section (for tools that produce output beyond the call itself) -->
      {#if hasResult && tc.name !== 'read'}
        <div class="border-t border-ctp-surface0/50"></div>
        <div class="p-3 text-xs overflow-x-auto" style="background: {resultIsError ? 'color-mix(in srgb, #e95f59 8%, #ffffff)' : '#f6f6f6'};">
          {#if tc.resultLanguage}
            <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[400px] overflow-y-auto">
              {@html resultHTML}
            </pre>
          {:else}
            <pre class="font-mono text-[11px] whitespace-pre-wrap break-words max-h-[400px] overflow-y-auto">{escapeHTML(resultContent)}</pre>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}
