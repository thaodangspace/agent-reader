<script>
  import { escapeHTML } from '$lib/utils/markdown.js';

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
    const oldLines = oldText.split('\n');
    const newLines = newText.split('\n');

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
   * Compute inline character-level diff between two lines.
   * Returns { prefix, oldMiddle, newMiddle, suffix } — all HTML-escaped.
   */
  function computeInlineParts(oldLine, newLine) {
    if (oldLine === undefined) return { prefix: escapeHTML(newLine), oldMiddle: '', newMiddle: '', suffix: '' };
    if (newLine === undefined) return { prefix: escapeHTML(oldLine), oldMiddle: '', newMiddle: '', suffix: '' };
    if (oldLine === newLine) return { prefix: escapeHTML(oldLine), oldMiddle: '', newMiddle: '', suffix: '' };

    // Find common prefix
    let prefixLen = 0;
    const minLen = Math.min(oldLine.length, newLine.length);
    while (prefixLen < minLen && oldLine[prefixLen] === newLine[prefixLen]) {
      prefixLen++;
    }

    // Find common suffix (don't overlap with prefix)
    let suffixLen = 0;
    while (
      suffixLen < minLen - prefixLen &&
      oldLine[oldLine.length - 1 - suffixLen] === newLine[newLine.length - 1 - suffixLen]
    ) {
      suffixLen++;
    }

    return {
      prefix: escapeHTML(oldLine.slice(0, prefixLen)),
      oldMiddle: escapeHTML(oldLine.slice(prefixLen, oldLine.length - (suffixLen || 0))),
      newMiddle: escapeHTML(newLine.slice(prefixLen, newLine.length - (suffixLen || 0))),
      suffix: escapeHTML(oldLine.slice(oldLine.length - (suffixLen || 0))),
    };
  }

  /**
   * Render the old line with deletions highlighted.
   */
  function renderOldLine(oldLine, newLine) {
    const p = computeInlineParts(oldLine, newLine);
    if (p.oldMiddle) {
      return `${p.prefix}<del class="diff-del">${p.oldMiddle}</del>${p.suffix}`;
    }
    return p.prefix + p.suffix;
  }

  /**
   * Render the new line with insertions highlighted.
   */
  function renderNewLine(oldLine, newLine) {
    const p = computeInlineParts(oldLine, newLine);
    if (p.newMiddle) {
      return `${p.prefix}<ins class="diff-ins">${p.newMiddle}</ins>${p.suffix}`;
    }
    return p.prefix + p.suffix;
  }
</script>

<div class="rounded-lg overflow-hidden border border-ctp-surface0 mb-2"
  style="background:color-mix(in srgb, #a6e3a1 8%, #1e1e2e)">
  <!-- Header -->
  <button
    class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs cursor-pointer"
    onclick={toggle}
  >
    <span
      class="transition-transform duration-200 text-[10px]"
      style="transform: {collapsed ? '' : 'rotate(90deg)'}"
    >▶</span>
    <span>📝</span>
    <span class="font-semibold" style="color:#a6e3a1">edit</span>
    <span class="text-ctp-overlay0 text-[10px] ml-auto truncate max-w-[300px]" title={filePath}>
      {filePath.split('/').slice(-2).join('/')}
    </span>
  </button>

  <!-- Diff content -->
  <div class="border-t border-ctp-surface0" class:hidden={collapsed}>
    <div class="text-[11px] font-mono" style="background:color-mix(in srgb, #1e1e2e 50%, #11111b);">
      {#each edits as edit, ei}
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
                  <div class="diff-line diff-line-removed flex">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0 select-none"
                      style="background:color-mix(in srgb, #f38ba8 12%, #11111b)">
                      {pair.oldLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #f38ba8 12%, #11111b); color:#f38ba8">-</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #f38ba8 12%, #11111b)">
                      {@html renderOldLine(pair.oldText, pair.newText)}
                    </span>
                  </div>
                  <div class="diff-line diff-line-added flex">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0 select-none"
                      style="background:color-mix(in srgb, #a6e3a1 12%, #11111b)">
                      {pair.newLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #a6e3a1 12%, #11111b); color:#a6e3a1">+</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #a6e3a1 12%, #11111b)">
                      {@html renderNewLine(pair.oldText, pair.newText)}
                    </span>
                  </div>
                {:else if pair.oldText !== undefined}
                  <div class="diff-line diff-line-removed flex">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0 select-none"
                      style="background:color-mix(in srgb, #f38ba8 12%, #11111b)">
                      {pair.oldLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #f38ba8 12%, #11111b); color:#f38ba8">-</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #f38ba8 12%, #11111b)">
                      {escapeHTML(pair.oldText)}
                    </span>
                  </div>
                {:else}
                  <div class="diff-line diff-line-added flex">
                    <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0 select-none"
                      style="background:color-mix(in srgb, #a6e3a1 12%, #11111b)">
                      {pair.newLine}
                    </span>
                    <span class="w-5 shrink-0 select-none"
                      style="background:color-mix(in srgb, #a6e3a1 12%, #11111b); color:#a6e3a1">+</span>
                    <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                      style="background:color-mix(in srgb, #a6e3a1 12%, #11111b)">
                      {escapeHTML(pair.newText)}
                    </span>
                  </div>
                {/if}
              {/each}
            {:else if segment.type === 'context'}
              <div class="diff-line diff-line-context flex">
                <span class="diff-line-num w-10 text-right pr-2 shrink-0 text-ctp-overlay0/60 select-none"
                  style="background:color-mix(in srgb, #1e1e2e 50%, #11111b)">
                  {segment.oldLine}
                </span>
                <span class="w-5 shrink-0 select-none"
                  style="background:color-mix(in srgb, #1e1e2e 50%, #11111b); color:#585b70"> </span>
                <span class="flex-1 pr-3 whitespace-pre overflow-x-auto"
                  style="background:color-mix(in srgb, #1e1e2e 50%, #11111b)">
                  {escapeHTML(segment.text)}
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
  .diff-del {
    background: color-mix(in srgb, #f38ba8 35%, transparent);
    text-decoration: none;
  }
  .diff-ins {
    background: color-mix(in srgb, #a6e3a1 35%, transparent);
  }
  .diff-line {
    line-height: 1.5;
  }
  .diff-line-num {
    user-select: none;
    opacity: 0.6;
  }
</style>
