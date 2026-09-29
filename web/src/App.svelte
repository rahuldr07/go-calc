<script lang="ts">
  import DetailLayer from "./lib/components/DetailLayer.svelte"
  import Dust from "./lib/components/Dust.svelte"
  import Foot from "./lib/components/Foot.svelte"
  import Head from "./lib/components/Head.svelte"
  import Keypad from "./lib/components/Keypad.svelte"
  import Notepad from "./lib/components/Notepad.svelte"
  import Terminal from "./lib/components/Terminal.svelte"
  import { setMode, view } from "./lib/state/view.svelte"

  let notepad: Notepad | undefined = $state()

  $effect(() => {
    document.documentElement.dataset.theme = view.theme
  })

  $effect(() => {
    document.body.classList.toggle("layer-on", view.layer && view.mode === "notepad")
    document.body.classList.toggle("pad-on", view.keypad && view.mode === "notepad")
  })

  function shortcuts(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key === ".") {
      event.preventDefault()
      setMode(view.mode === "notepad" ? "terminal" : "notepad")
    }
  }
</script>

<svelte:window onkeydown={shortcuts} />

<Dust />

<div class="shell">
  <Head />
  {#if view.mode === "notepad"}
    <Notepad bind:this={notepad} />
  {:else}
    <Terminal />
  {/if}
  <Foot />
</div>

{#if view.keypad && view.mode === "notepad"}
  <Keypad
    insert={(text) => notepad?.insert(text)}
    press={(command) => notepad?.press(command)}
  />
{/if}

<DetailLayer />
