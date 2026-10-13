<script lang="ts">
  import { setAngle, setDigits, sheet } from "../state/sheet.svelte"
  import { encodeSheet, linkFor } from "../state/share"

  let shareWord = $state("share")
  let resetShare: ReturnType<typeof setTimeout> | undefined

  async function share() {
    const hash = await encodeSheet(sheet.source)
    try {
      await navigator.clipboard.writeText(linkFor(hash))
      shareWord = "link copied"
    } catch {
      history.replaceState(null, "", hash)
      shareWord = "link in address bar"
    }
    clearTimeout(resetShare)
    resetShare = setTimeout(() => (shareWord = "share"), 2400)
  }
  import { setMode, toggleHelp, toggleKeypad, view } from "../state/view.svelte"
  import { current, setTheme } from "../theme/theme.svelte"
  import { THEME_LIST } from "../theme/themes"

  const engineWord = $derived(
    sheet.engineStatus === "ready"
      ? "engine ready"
      : sheet.engineStatus === "loading"
        ? "starting engine"
        : `engine failed — ${sheet.failure}`,
  )
</script>

<header class="topbar">
  <div class="brand" data-state={sheet.engineStatus}>
    <b>go-calc</b>
    <span class="state" role="status"><span class="dot" aria-hidden="true"></span>{engineWord}</span>
  </div>

  <div class="pills" role="group" aria-label="Surface">
    <button type="button" aria-pressed={view.mode === "notepad"} onclick={() => setMode("notepad")}>Notepad</button>
    <button type="button" aria-pressed={view.mode === "terminal"} onclick={() => setMode("terminal")}>Terminal</button>
  </div>

  <div class="tools">
    <span class="chip digits" title="Significant digits shown">
      <button class="step" type="button" aria-label="Fewer digits" onclick={() => void setDigits(sheet.digits - 1)}>−</button>
      <b>{sheet.digits}</b> digits
      <button class="step" type="button" aria-label="More digits" onclick={() => void setDigits(sheet.digits + 1)}>+</button>
    </span>
    <button class="chip angle" type="button" title="Angle unit for trigonometry" onclick={() => void setAngle(sheet.angle === "rad" ? "deg" : "rad")}>{sheet.angle}</button>
    <button class="chip keypad" type="button" aria-pressed={view.keypad} onclick={toggleKeypad}>keypad</button>
    <button class="chip share" type="button" title="Copy a link that opens this sheet" disabled={sheet.source.trim() === ""}
      onclick={() => void share()}>{shareWord}</button>
    <button class="chip help-chip" type="button" aria-pressed={view.help} title="Help (Shift + ?)" onclick={toggleHelp}>help</button>
    <span class="swatches" role="group" aria-label="Theme">
      {#each THEME_LIST as entry (entry.id)}
        <button type="button" class="swatch" aria-pressed={current().id === entry.id} aria-label={`${entry.label} theme`}
          title={`${entry.label} — ${entry.note}`} style={`--a:${entry.colors.paper};--b:${entry.colors.accent}`}
          onclick={() => setTheme(entry.id)}></button>
      {/each}
    </span>
  </div>
</header>
