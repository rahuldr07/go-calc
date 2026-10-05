<script lang="ts">
  import { setAngle, setDigits, sheet } from "../state/sheet.svelte"
  import { setMode, toggleKeypad, view } from "../state/view.svelte"
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
    <span class="chip">
      <button class="step" type="button" aria-label="Fewer digits" onclick={() => void setDigits(sheet.digits - 1)}>−</button>
      <b>{sheet.digits}</b> digits
      <button class="step" type="button" aria-label="More digits" onclick={() => void setDigits(sheet.digits + 1)}>+</button>
    </span>
    <button class="chip" type="button" onclick={() => void setAngle(sheet.angle === "rad" ? "deg" : "rad")}>{sheet.angle}</button>
    <button class="chip" type="button" aria-pressed={view.keypad} onclick={toggleKeypad}>keypad</button>
    <span class="chip theme">
      {#each THEME_LIST as entry (entry.id)}
        <button type="button" aria-pressed={current().id === entry.id} title={entry.note} onclick={() => setTheme(entry.id)}
          >{current().id === entry.id ? entry.label : entry.label.slice(0, 1)}</button>
      {/each}
    </span>
  </div>
</header>
