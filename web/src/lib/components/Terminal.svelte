<script lang="ts">
  import { onMount } from "svelte"
  import type { Line } from "../engine/schema"
  import { located } from "../editor/notepad"
  import { setSource, sheet } from "../state/sheet.svelte"

  let entry = $state("")
  let scroller: HTMLElement | undefined = $state()
  let field: HTMLInputElement | undefined = $state()
  let recall = $state(-1)

  const rows = $derived(
    sheet.source
      .split("\n")
      .map((src, n) => ({ src, n, line: sheet.lines.find((l) => l.n === n) }))
      .filter((r) => r.src.trim() !== ""),
  )

  const typed = $derived(rows.filter((r) => !r.src.trim().startsWith("#")).map((r) => r.src))

  function answerOf(line: Line | undefined) {
    if (!line) return ""
    return line.status === "error" ? located(line.error ?? "refused") : line.text
  }

  function extras(line: Line | undefined): string[] {
    if (!line) return []
    return [
      ...(line.notes ?? []).map((n) => n.text),
      ...(line.qualifiers ?? []),
      ...(line.warning ? [line.warning] : []),
    ]
  }

  function bottom() {
    queueMicrotask(() => scroller?.scrollTo({ top: scroller.scrollHeight }))
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    const text = entry.trim()
    if (text === "") return
    entry = ""
    recall = -1
    await setSource(sheet.source.trim() === "" ? text : `${sheet.source.replace(/\n+$/, "")}\n${text}`)
    bottom()
  }

  function history(event: KeyboardEvent) {
    if (event.key !== "ArrowUp" && event.key !== "ArrowDown") return
    if (typed.length === 0) return
    event.preventDefault()
    const step = event.key === "ArrowUp" ? 1 : -1
    recall = Math.max(-1, Math.min(typed.length - 1, recall + step))
    entry = recall === -1 ? "" : typed[typed.length - 1 - recall]
  }

  onMount(() => {
    field?.focus()
    bottom()
  })
</script>

<div class="term" bind:this={scroller}>
  {#if rows.length === 0}
    <p class="term-empty">Type an expression below and press Enter. ↑ recalls earlier lines.</p>
  {/if}
  {#each rows as row (row.n)}
    {#if row.src.trim().startsWith("#")}
      <div class="term-comment">{row.src}</div>
    {:else}
      <div class="term-row" data-status={row.line?.status}>
        <div class="term-in">{row.src}</div>
        <div class="term-out">
          <b>{answerOf(row.line)}</b>
          {#if row.line}<span class="term-status">{row.line.status}</span>{/if}
          {#each extras(row.line) as extra}<span class="term-extra">{extra}</span>{/each}
        </div>
      </div>
    {/if}
  {/each}
</div>
<form class="prompt" onsubmit={submit}>
  <span class="prompt-mark" aria-hidden="true">›</span>
  <input bind:this={field} bind:value={entry} onkeydown={history} aria-label="Terminal input" autocomplete="off"
    spellcheck="false" placeholder="type an expression" />
</form>
