<script>
  import { escapeHTML } from '$lib/utils/markdown.js';
  import { ChevronRight, ChevronDown, FilePenLine } from '@lucide/svelte';

  let { filePath, edits } = $props();

  let collapsed = $state(false);

  function toggle() {
    collapsed = !collapsed;
  }

  /**
   * Compute a line-level unified diff between oldText and newText.
   * Returns an array of segments:
   *   { type: 'context', text, oldLine, newLine }
   *   { type: 'ellipsis', count }
   *   { type: 'changed', pairs: [{ oldText, newText?, oldLine, newLine? }] }
   *
   * For changed segments, consecutive removed+added lines are paired
   * for inline character-level diff display.
   */
  function computeDiff(oldText, newText) {
    const oldLines = (oldText ?? '').split('\n');
    const newLines = (newText ?? '').split('\n');

    // Simple LCS-based diff
    const m = oldLines.length;
    const n = newLines.length;

    // Build LCS table
    const dp = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0));
    for (let i = 1; i <= m; i++) {
      for (let j = 1; j <= n; j++) {
        if (oldLines[i - 1] === newLines[j - 1]) {
          dp[i][j] = dp[i - 1][j - 1] + 1;
        } else {
          dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
        }
      }
    }

    // Backtrack to produce ops
    let i = m, j = n;
    const ops = [];

    while (i > 0 || j > 0) {
      if (i > 0 && j > 0 && oldLines[i - 1] === newLines[j - 1]) {
        ops.push({ type: 'context', oldLine: i, newLine: j, text: oldLines[i - 1] });
        i--; j--;
      } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
        ops.push({ type: 'added', newLine: j, text: newLines[j - 1] });
        j--;
      } else if (i > 0) {
        ops.push({ type: 'removed', oldLine: i, text: oldLines[i - 1] });
        i--;
      }
    }

    ops.reverse();

    // Group into segments with paired changes for inline diff
    const MAX_CONTEXT = 3;
    const segments = [];
    let removedBuf = [];
    let addedBuf = [];
    let contextBuf = [];

    function flushChangePair() {
      if (removedBuf.length === 0 && addedBuf.length === 0) return;

      // Pair removed and added lines for inline diff
      const pairs = [];
      const maxLen = Math.max(removedBuf.length, addedBuf.length);
      for (let k = 0; k < maxLen; k++) {
        pairs.push({
          oldText: k < removedBuf.length ? removedBuf[k].text : undefined,
          newText: k < addedBuf.length ? addedBuf[k].text : undefined,
          oldLine: k < removedBuf.length ? removedBuf[k].oldLine : undefined,
          newLine: k < addedBuf.length ? addedBuf[k].newLine : undefined,
        });
      }

      segments.push({ type: 'changed', pairs });
      removedBuf = [];
      addedBuf = [];
    }

    function flushContext() {
      if (contextBuf.length === 0) return;
      if (contextBuf.length > MAX_CONTEXT) {
        const truncated = contextBuf.length - MAX_CONTEXT;
        segments.push({ type: 'ellipsis', count: truncated });
        segments.push(...contextBuf.slice(-MAX_CONTEXT).map(c => ({
          type: 'context',
          text: c.text,
          oldLine: c.oldLine,
          newLine: c.newLine,
        })));
      } else {
        segments.push(...contextBuf.map(c => ({
          type: 'context',
          text: c.text,
          oldLine: c.oldLine,
          newLine: c.newLine,
        })));
      }
      contextBuf = [];
    }

    for (const op of ops) {
      if (op.type === 'context') {
        // If we have buffered changes, flush them first
        if (removedBuf.length > 0 || addedBuf.length > 0) {
          flushChangePair();
          // Context after a change goes directly to buffer
          contextBuf.push(op);
        } else {
          contextBuf.push(op);
        }
      } else {
        // Flush any pending context before a change
        if (contextBuf.length > 0) {
          flushContext();
        }
        if (op.type === 'removed') {
          removedBuf.push(op);
        } else if (op.type === 'added') {
          addedBuf.push(op);
        }
      }
    }

    // Flush remaining
    flushChangePair();
    flushContext();

    return segments;
  }

  /**
   * Escape code for HTML display.
   */
  function highlight(line) {
    return escapeHTML(line || '');
  }

  /**
   * Given highlighted HTML and a raw text character range [start, end),
   * extract the corresponding HTML by counting only raw characters.
   */
  function extractHighlightedRange(hlHtml, start, end) {
    let pos = 0;
    let i = 0;
    const len = hlHtml.length;

    // Skip to start
    while (i < len && pos < start) {
      if (hlHtml[i] === '<') {
        // Skip entire tag
        while (i < len && hlHtml[i] !== '>') i++;
        i++;
      } else {
        // Check for &amp; &lt; &gt; entities
        if (hlHtml.substring(i, i + 5) === '&amp;' || hlHtml.substring(i, i + 4) === '&lt;' || hlHtml.substring(i, i + 4) === '&gt;' || hlHtml.substring(i, i + 6) === '&#39;' || hlHtml.substring(i, i + 6) === '&quot;') {
          const semi = hlHtml.indexOf(';', i);
          if (semi !== -1) i = semi + 1;
          else i++;
        } else {
          i++;
        }
        pos++;
      }
    }
    const rangeStart = i;

    // Find end
    while (i < len && pos < end) {
      if (hlHtml[i] === '<') {
        while (i < len && hlHtml[i] !== '>') i++;
        i++;
      } else {
        if (hlHtml.substring(i, i + 5) === '&amp;' || hlHtml.substring(i, i + 4) === '&lt;' || hlHtml.substring(i, i + 4) === '&gt;' || hlHtml.substring(i, i + 6) === '&#39;' || hlHtml.substring(i, i + 6) === '&quot;') {
          const semi = hlHtml.indexOf(';', i);
          if (semi !== -1) i = semi + 1;
          else i++;
        } else {
          i++;
        }
        pos++;
      }
    }
    const rangeEnd = i;

    // Now we need to include any open tags from rangeStart through rangeEnd
    // Find the tag context at rangeStart
    let tagContext = '';
    let j = 0;
    while (j < rangeStart) {
      if (hlHtml[j] === '<' && (j + 1 >= hlHtml.length || hlHtml[j + 1] !== '/')) {
        // Opening tag - extract it
        let endTag = hlHtml.indexOf('>', j);
        if (endTag !== -1 && endTag < rangeStart) {
          tagContext = hlHtml.substring(j, endTag + 1);
          j = endTag + 1;
        } else break;
      } else {
        j++;
      }
    }

    // Collect all closing tags we need between rangeEnd and end of their tags
    let closingTags = '';
    let k = rangeEnd;
    // Find any open tags that started before rangeEnd but haven't closed
    const openStack = [];
    j = 0;
    while (j < rangeEnd) {
      if (hlHtml[j] === '<') {
        if (hlHtml[j + 1] === '/') {
          // Closing tag - pop
          openStack.pop();
          let endTag = hlHtml.indexOf('>', j);
          j = endTag + 1;
        } else {
          // Opening tag
          let endTag = hlHtml.indexOf('>', j);
          if (endTag !== -1) {
            const tag = hlHtml.substring(j, endTag + 1);
            if (endTag < rangeStart) {
              // This tag started before our range, we need to reopen it
              openStack.push(tag);
            }
            j = endTag + 1;
          } else break;
        }
      } else {
        j++;
      }
    }

    // Build result: reopen tags + content + close tags
    let result = '';
    for (const tag of openStack) {
      result += tag;
    }
    result += hlHtml.substring(rangeStart, rangeEnd);
    for (let c = openStack.length - 1; c >= 0; c--) {
      const tagName = openStack[c].match(/<(\w+)/);
      if (tagName) result += '</' + tagName[1] + '>';
    }
    return result;
  }

  /**
   * Render a line with syntax highlighting and optional inline diff.
   * For context lines: just highlight.
   * For changed lines: highlight + wrap changed portion.
   */
  function renderHighlightedLine(rawLine, changedStart, changedEnd, className) {
    const hl = highlight(rawLine);
    if (changedStart === undefined || changedEnd === undefined || changedStart >= changedEnd) {
      return hl;
    }
    const before = extractHighlightedRange(hl, 0, changedStart);
    const changed = extractHighlightedRange(hl, changedStart, changedEnd);
    const after = extractHighlightedRange(hl, changedEnd, rawLine.length);
    return before + '<span class="' + className + '">' + changed + '</span>' + after;
  }

  /**
   * Render the old line with syntax highlighting and deletions marked.
   */
  function renderOldLine(oldLine, newLine) {
    if (oldLine === undefined) return '';
    if (newLine === undefined) {
      // Pure deletion, highlight whole line
      return highlight(oldLine);
    }
    if (oldLine === newLine) return highlight(oldLine);

    // Find common prefix length in raw text
    let prefixLen = 0;
    const minLen = Math.min(oldLine.length, newLine.length);
    while (prefixLen < minLen && oldLine[prefixLen] === newLine[prefixLen]) {
      prefixLen++;
    }

    // Find common suffix length
    let suffixLen = 0;
    while (
      suffixLen < minLen - prefixLen &&
      oldLine[oldLine.length - 1 - suffixLen] === newLine[newLine.length - 1 - suffixLen]
    ) {
      suffixLen++;
    }

    return renderHighlightedLine(oldLine, prefixLen, oldLine.length - (suffixLen || 0), 'diff-del-inline');
  }

  /**
   * Render the new line with syntax highlighting and insertions marked.
   */
  function renderNewLine(oldLine, newLine) {
    if (newLine === undefined) return '';
    if (oldLine === undefined) {
      // Pure addition, highlight whole line
      return highlight(newLine);
    }
    if (oldLine === newLine) return highlight(oldLine);

    let prefixLen = 0;
    const minLen = Math.min(oldLine.length, newLine.length);
    while (prefixLen < minLen && oldLine[prefixLen] === newLine[prefixLen]) {
      prefixLen++;
    }

    let suffixLen = 0;
    while (
      suffixLen < minLen - prefixLen &&
      oldLine[oldLine.length - 1 - suffixLen] === newLine[newLine.length - 1 - suffixLen]
    ) {
      suffixLen++;
    }

    return renderHighlightedLine(newLine, prefixLen, newLine.length - (suffixLen || 0), 'diff-ins-inline');
  }
</script>

<div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2"
  style="background:color-mix(in srgb, #65b73b 8%, #ffffff)">
  <!-- Header -->
  <button
    class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer"
    onclick={toggle}
  >
    <span class="flex items-center">
      {#if collapsed}
        <ChevronRight size={12} />
      {:else}
        <ChevronDown size={12} />
      {/if}
    </span>
    <FilePenLine size={14} class="text-[#65b73b]" />
    <span class="font-semibold" style="color:#65b73b">edit</span>
    <span class="text-ctp-overlay0 text-[10px] ml-auto truncate max-w-[300px]" title={filePath}>
      {(filePath ?? '').split('/').slice(-2).join('/')}
    </span>
  </button>

  <!-- Diff content -->
  <div class="border-t border-ctp-surface0" class:hidden={collapsed}>
    <div class="text-[11px] font-mono" style="background:color-mix(in srgb, #ffffff 50%, #ffffff);">
      {#each (edits ?? []) as edit, ei}
        {#if ei > 0}
          <div class="border-t border-ctp-surface0/50"></div>
        {/if}
        <div class="diff-block">
          {#each computeDiff(edit.oldText, edit.newText) as segment}
            {#if segment.type === 'ellipsis'}
              <div class="px-3 py-0.5 text-ctp-overlay0 italic text-[10px] select-none">
                … {segment.count} unchanged lines …
              </div>
            {:else if segment.type === 'changed'}
              {#each segment.pairs as pair}
                {#if pair.oldText !== undefined && pair.newText !== undefined}
                  <div class="diff-line diff-line-removed flex leading-normal">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0/60 select-none"
                      style="background:color-mix(in srgb, #e95f59 12%, #ffffff)">
                      {pair.oldLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #e95f59 12%, #ffffff); color:#e95f59">-</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #e95f59 12%, #ffffff)">
                      {@html renderOldLine(pair.oldText, pair.newText)}
                    </span>
                  </div>
                  <div class="diff-line diff-line-added flex leading-normal">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0/60 select-none"
                      style="background:color-mix(in srgb, #65b73b 12%, #ffffff)">
                      {pair.newLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #65b73b 12%, #ffffff); color:#65b73b">+</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #65b73b 12%, #ffffff)">
                      {@html renderNewLine(pair.oldText, pair.newText)}
                    </span>
                  </div>
                {:else if pair.oldText !== undefined}
                  <div class="diff-line diff-line-removed flex leading-normal">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0/60 select-none"
                      style="background:color-mix(in srgb, #e95f59 12%, #ffffff)">
                      {pair.oldLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #e95f59 12%, #ffffff); color:#e95f59">-</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #e95f59 12%, #ffffff)">
                      {@html highlight(pair.oldText)}
                    </span>
                  </div>
                {:else}
                  <div class="diff-line diff-line-added flex leading-normal">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0/60 select-none"
                      style="background:color-mix(in srgb, #65b73b 12%, #ffffff)">
                      {pair.newLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #65b73b 12%, #ffffff); color:#65b73b">+</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #65b73b 12%, #ffffff)">
                      {@html highlight(pair.newText)}
                    </span>
                  </div>
                {/if}
              {/each}
            {:else if segment.type === 'context'}
              <div class="diff-line diff-line-context flex leading-normal">
                <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0/60 select-none"
                  style="background:color-mix(in srgb, #ffffff 50%, #ffffff)">
                  {segment.oldLine}
                </span>
                <span class="w-5 shrink-0 select-none"
                  style="background:color-mix(in srgb, #ffffff 50%, #ffffff); color:#777777"> </span>
                <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                  style="background:color-mix(in srgb, #ffffff 50%, #ffffff)">
                  {@html highlight(segment.text)}
                </span>
              </div>
            {/if}
          {/each}
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  .diff-del,
  .diff-del-inline {
    background: color-mix(in srgb, #e95f59 25%, transparent);
    text-decoration: none;
  }
  .diff-ins,
  .diff-ins-inline {
    background: color-mix(in srgb, #65b73b 25%, transparent);
  }
</style>
