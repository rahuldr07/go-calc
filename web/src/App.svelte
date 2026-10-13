<script lang="ts">
  import Detail from "./lib/components/Detail.svelte"
  import Help from "./lib/components/Help.svelte"
  import Keypad from "./lib/components/Keypad.svelte"
  import Sheet from "./lib/components/Sheet.svelte"
  import Terminal from "./lib/components/Terminal.svelte"
  import TopBar from "./lib/components/TopBar.svelte"
  import { apply, theme } from "./lib/theme/theme.svelte"
  import { onMount } from "svelte"
  import { openShared, setSource, sheet } from "./lib/state/sheet.svelte"
  import { decodeSheet, sharedHash } from "./lib/state/share"
  import { setMode, toggleHelp, view } from "./lib/state/view.svelte"

  let sheetPane: Sheet | undefined = $state()

  $effect(() => {
    void theme.id
    apply()
  })

  onMount(() => {
    const hash = sharedHash()
    if (hash === "") return
    void decodeSheet(hash).then((text) => {
      history.replaceState(null, "", location.pathname + location.search)
      if (text !== null) void openShared(text)
    })
  })

  function tryIt(expression: string) {
    const kept = sheet.source.replace(/\n+$/, "")
    void setSource(kept === "" ? expression : `${kept}\n${expression}`)
    setMode("notepad")
    toggleHelp()
  }

  function shortcuts(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key === ".") {
      event.preventDefault()
      setMode(view.mode === "notepad" ? "terminal" : "notepad")
    }
    if (event.key === "?" && event.shiftKey && !(event.target as HTMLElement)?.closest(".cm-editor")) {
      event.preventDefault()
      toggleHelp()
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
      {#if view.mode === "notepad"}<Detail />{/if}
      {#if view.keypad && view.mode === "notepad"}
        <Keypad insert={(t) => sheetPane?.insert(t)} press={(c) => sheetPane?.press(c)} />
      {/if}
    </section>
  </div>
  </div>
 </div>
 {#if view.help}<Help close={toggleHelp} {tryIt} />{/if}
</div>
