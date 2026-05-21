<script>
  import { escapeHTML, highlightCode } from '$lib/utils/markdown.js';
  import { detectLanguageFromPath } from '$lib/utils/language.js';
  import { unescapeJsonString } from '$lib/utils/json.js';
  import DiffView from './DiffView.svelte';
  import { ChevronRight, ChevronDown, FileText, BookOpen, Terminal, Wrench } from '@lucide/svelte';

  let { tc } = $props();
  let collapsed = $state(true);

  $effect(() => {
    collapsed = tc.name?.toLowerCase() === 'askuserquestion' ? (tc.result !== undefined && tc.result !== null) : true;
  });

  // Parse arguments for structured display
  let argsStr = $derived(
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

  let answersMap = $derived.by(() => {
    if (!resultContent) return {};
    const answers = {};
    const regex = /"([^"]+)"="([^"]+)"/g;
    let match;
    try {
      while ((match = regex.exec(resultContent)) !== null) {
        const q = match[1].trim().toLowerCase();
        const a = match[2].trim().toLowerCase();
        answers[q] = a;
      }
    } catch {}
    return answers;
  });

  function checkSelected(questionText, optionLabel) {
    if (!hasResult || !resultContent) return false;
    
    const qTextClean = questionText.trim().toLowerCase();
    const optLabelClean = optionLabel.trim().toLowerCase();
    
    // Check in parsed answers map
    for (const [q, a] of Object.entries(answersMap)) {
      if (qTextClean.includes(q) || q.includes(qTextClean)) {
        if (a === optLabelClean) {
          return true;
        }
      }
    }
    
    // Direct string substring check fallback
    try {
      const labelEscaped = optionLabel.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&');
      const directRegex = new RegExp(`=["']${labelEscaped}["']`, 'i');
      return directRegex.test(resultContent);
    } catch {
      return false;
    }
  }

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
{:else if tc.name?.toLowerCase() === 'askuserquestion' && parsedArgs && Array.isArray(parsedArgs.questions)}
  <div class="rounded-xl overflow-hidden border mb-2 shadow-sm transition-all duration-300 w-full text-left"
       style="background: var(--color-ctp-base); border-color: {hasResult ? 'color-mix(in srgb, var(--color-ctp-green) 35%, var(--color-ctp-crust))' : 'color-mix(in srgb, var(--color-ctp-blue) 35%, var(--color-ctp-crust))'};">
    
    <!-- Header -->
    <button class="w-full flex items-center gap-2 px-3 py-2 text-xs font-medium cursor-pointer transition-colors duration-150 text-left"
            style="background: {hasResult ? 'color-mix(in srgb, var(--color-ctp-green) 8%, var(--color-ctp-base))' : 'color-mix(in srgb, var(--color-ctp-blue) 8%, var(--color-ctp-base))'};"
            onclick={toggle}>
      <span class="flex items-center text-ctp-overlay0 shrink-0">
        {#if collapsed}
          <ChevronRight size={12} />
        {:else}
          <ChevronDown size={12} />
        {/if}
      </span>
      <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" 
           stroke={hasResult ? 'var(--color-ctp-green)' : 'var(--color-ctp-blue)'} 
           stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
        <line x1="12" y1="17" x2="12.01" y2="17"></line>
      </svg>
      <span class="font-bold text-xs shrink-0" style="color: {hasResult ? 'var(--color-ctp-green)' : 'var(--color-ctp-blue)'};">
        AskUserQuestion
      </span>
      
      {#if collapsed}
        {#if parsedArgs.questions?.[0]?.question}
          <span class="text-ctp-overlay0 text-[10px] ml-auto truncate max-w-[280px] font-normal" title={parsedArgs.questions[0].question}>
            {parsedArgs.questions[0].question}
          </span>
        {:else}
          <span class="text-ctp-overlay0 text-[10px] ml-auto font-normal">Show questions</span>
        {/if}
      {:else}
        <span class="ml-auto">
          {#if hasResult}
            <span class="px-2 py-0.5 rounded-full text-[9px] font-bold text-white bg-ctp-green" style="background: var(--color-ctp-green)">
              Answered
            </span>
          {:else}
            <span class="px-2 py-0.5 rounded-full text-[9px] font-bold text-white bg-ctp-blue animate-pulse" style="background: var(--color-ctp-blue)">
              Pending
            </span>
          {/if}
        </span>
      {/if}
    </button>

    <!-- Content -->
    <div class="border-t border-ctp-crust" class:hidden={collapsed}>
      <div class="p-4 flex flex-col gap-5 bg-ctp-base">
        {#each parsedArgs.questions as q, qIdx}
          <div class="flex flex-col gap-2">
            {#if q.header}
              <div class="text-[10px] font-bold tracking-wider uppercase text-ctp-overlay0">
                {q.header}
              </div>
            {/if}
            <div class="text-sm font-semibold text-ctp-text leading-snug">
              {q.question}
            </div>

            <!-- Options -->
            {#if Array.isArray(q.options)}
              <div class="grid grid-cols-1 gap-2 mt-1">
                {#each q.options as opt}
                  {@const isSelected = checkSelected(q.question, opt.label)}
                  <div class="p-3 rounded-xl border text-xs transition-all duration-200"
                       style="background: {isSelected ? 'color-mix(in srgb, var(--color-ctp-green) 6%, var(--color-ctp-base))' : 'var(--color-ctp-mantle)'};
                              border-color: {isSelected ? 'var(--color-ctp-green)' : 'var(--color-ctp-crust)'};
                              border-width: {isSelected ? '2px' : '1px'};">
                    <div class="flex items-start gap-3">
                      <div class="mt-0.5 shrink-0">
                        {#if isSelected}
                          <div class="w-4 h-4 rounded-full bg-ctp-green flex items-center justify-center text-white" style="background: var(--color-ctp-green)">
                            <svg xmlns="http://www.w3.org/2000/svg" class="w-2.5 h-2.5" viewBox="0 0 24 24" fill="none" 
                                 stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round">
                              <polyline points="20 6 9 17 4 12"></polyline>
                            </svg>
                          </div>
                        {:else}
                          <div class="w-4 h-4 rounded-full border border-ctp-surface1 bg-ctp-base"></div>
                        {/if}
                      </div>
                      <div class="flex flex-col gap-1">
                        <span class="font-bold text-ctp-text text-[13px]">{opt.label}</span>
                        {#if opt.description}
                          <span class="text-ctp-overlay0 leading-normal text-[11px] font-normal">{opt.description}</span>
                        {/if}
                      </div>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <!-- Result Footer -->
      {#if hasResult}
        <div class="border-t px-4 py-3 text-xs bg-ctp-mantle border-ctp-crust">
          <div class="font-bold text-ctp-text mb-1.5 flex items-center gap-1.5">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-ctp-green shrink-0" viewBox="0 0 24 24" fill="none" 
                 stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
              <polyline points="22 4 12 14.01 9 11.01"></polyline>
            </svg>
            <span class="text-[10px] uppercase tracking-wider font-bold text-ctp-text">Selected Answers</span>
          </div>
          <div class="text-ctp-overlay1 leading-relaxed pl-5.5 text-[11px] font-medium font-mono whitespace-pre-wrap break-words">
            {resultContent}
          </div>
        </div>
      {/if}
    </div>
  </div>
{:else}
  <div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2"
       style="background: {hasResult && resultIsError ? 'color-mix(in srgb, #e95f59 8%, #ffffff)' : 'color-mix(in srgb, #135ce0 6%, #ffffff)'}">
    <button class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer text-left" onclick={toggle}>
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
