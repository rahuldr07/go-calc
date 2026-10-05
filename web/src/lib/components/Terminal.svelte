<script lang="ts">
  import type { Line } from "../engine/schema"
  import { setSource, sheet } from "../state/sheet.svelte"

  let entry = $state("")
  let scroller: HTMLElement | undefined = $state()

  const rows = $derived(
    sheet.source
      .split("\n")
      .map((src, n) => ({ src, line: sheet.lines.find((l) => l.n === n) }))
      .filter((r) => r.src.trim() !== ""),
  )

  function answerOf(line: Line | undefined) {
    if (!line) return ""
    return line.status === "error" ? (line.error ?? "refused") : line.text
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    const typed = entry.trim()
    if (typed === "") return
    entry = ""
    await setSource(sheet.source === "" ? typed : `${sheet.source}\n${typed}`)
    queueMicrotask(() => scroller?.scrollTo({ top: scroller.scrollHeight }))
  }
</script>

<div class="term" bind:this={scroller}>
  {#each rows as row (row.src + row.line?.n)}
    <div class="term-row">
      <div class="term-in">{row.src}</div>
      <div class="term-out">
        <b>{answerOf(row.line)}</b>
        {#if row.line}<span class="cap" style="margin-left:10px">{row.line.status}</span>{/if}
        {#each row.line?.notes ?? [] as note}<br />{note.text}{/each}
        {#if row.line?.warning}<br />{row.line.warning}{/if}
      </div>
    </div>
  {/each}
</div>
<form class="prompt" onsubmit={submit}>
  <input bind:value={entry} aria-label="Terminal input" autocomplete="off" spellcheck="false" placeholder="type an expression" />
</form>
