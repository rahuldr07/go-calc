<script lang="ts">
  import { setSource, sheet } from "../state/sheet.svelte"
  import type { Line } from "../engine/schema"

  let entry = $state("")
  let scroller: HTMLElement | undefined = $state()

  const transcript = $derived(
    sheet.source
      .split("\n")
      .map((src, n) => ({ src, line: sheet.lines.find((l) => l.n === n) }))
      .filter((row) => row.src.trim() !== ""),
  )

  function answerOf(line: Line | undefined): string {
    if (!line) return ""
    if (line.status === "error") return line.error ?? "refused"
    return line.text
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    const typed = entry.trim()
    if (typed === "") return
    entry = ""
    const next = sheet.source === "" ? typed : `${sheet.source}\n${typed}`
    await setSource(next)
    queueMicrotask(() => scroller?.scrollTo({ top: scroller.scrollHeight }))
  }
</script>

<main class="term" bind:this={scroller}>
  <div class="tb"><div class="out">go-calc · wasm · codata 2022 · the same document as the notepad</div></div>
  {#each transcript as row (row.src + row.line?.n)}
    <div class="tb">
      <div class="in">{row.src}</div>
      <div class="out"><b>{answerOf(row.line)}</b>
        {#if row.line}<span class="st-word">{row.line.status}</span>{/if}
        {#each row.line?.notes ?? [] as note}<br />{note.text}{/each}
        {#if row.line?.warning}<br />{row.line.warning}{/if}
      </div>
    </div>
  {/each}
  <form class="prompt" onsubmit={submit}>
    <input bind:value={entry} aria-label="Terminal input" autocomplete="off" spellcheck="false" placeholder="type an expression" />
  </form>
</main>
