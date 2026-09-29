<script lang="ts">
  import { Slider, ToggleGroup } from "bits-ui"
  import { setDigits, setAngle, sheet } from "../state/sheet.svelte"
  import { setMode, toggleKeypad, toggleTheme, view } from "../state/view.svelte"

  const engineWord = $derived(
    sheet.engineStatus === "ready" ? "ready" : sheet.engineStatus === "loading" ? "loading" : "failed",
  )
</script>

<header class="head">
  <div class="brand">go-calc<em></em></div>
  <nav class="sheets">
    <button aria-current="true">scratch</button>
  </nav>
  <div class="tools">
    <div class="keys">
      <ToggleGroup.Root type="single" value={view.mode} onValueChange={(v) => v && setMode(v as "notepad" | "terminal")}>
        <ToggleGroup.Item value="notepad" class="key">notepad</ToggleGroup.Item>
        <ToggleGroup.Item value="terminal" class="key">terminal</ToggleGroup.Item>
      </ToggleGroup.Root>
      <button class="key" type="button" onclick={toggleTheme}>{view.theme === "light" ? "dark" : "light"}</button>
      <button class="key" type="button" aria-pressed={view.keypad} onclick={toggleKeypad}>keypad</button>
    </div>
    <div class="knob" title="Digits">
      <Slider.Root
        type="single"
        min={1}
        max={60}
        step={1}
        value={sheet.digits}
        onValueChange={(v) => void setDigits(v)}
      >
        <span class="dial" aria-hidden="true"></span>
        <Slider.Thumb index={0} class="dial-grip" />
      </Slider.Root>
      <span>digits<br /><b style="color:var(--ivory);font-weight:400">{sheet.digits}</b></span>
    </div>
    <div class="lcd">
      <button type="button" onclick={() => void setAngle(sheet.angle === "rad" ? "deg" : "rad")}>{sheet.angle}</button>
      <span>core <b>{engineWord}</b></span>
      <span>recalc <b>{sheet.elapsed}ms</b></span>
    </div>
  </div>
</header>
