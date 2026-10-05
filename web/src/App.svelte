<script lang="ts">
  import Keypad from "./lib/components/Keypad.svelte"
  import Lens from "./lib/components/Lens.svelte"
  import Sheet from "./lib/components/Sheet.svelte"
  import Terminal from "./lib/components/Terminal.svelte"
  import TopBar from "./lib/components/TopBar.svelte"
  import { apply, theme } from "./lib/theme/theme.svelte"
  import { setMode, view } from "./lib/state/view.svelte"

  let sheetPane: Sheet | undefined = $state()

  $effect(() => {
    void theme.id
    apply()
  })

  function shortcuts(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key === ".") {
      event.preventDefault()
      setMode(view.mode === "notepad" ? "terminal" : "notepad")
    }
  }
</script>

<svelte:window onkeydown={shortcuts} />

<div class="stage">
 <div class="outer">
  <div class="app">
  <TopBar />
  <div class="workspace">
    <section class="sheet-pane" aria-label="Calculation sheet">
      {#if view.mode === "notepad"}
        <Sheet bind:this={sheetPane} />
      {:else}
        <Terminal />
      {/if}
      {#if view.keypad && view.mode === "notepad"}
        <Keypad insert={(t) => sheetPane?.insert(t)} press={(c) => sheetPane?.press(c)} />
      {/if}
    </section>
    <section class="lens-pane" aria-label="Analysis lens">
      <Lens />
    </section>
  </div>
  </div>
 </div>
</div>
