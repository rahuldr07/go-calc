<script lang="ts">
  import katex from "katex"
  import "katex/dist/katex.min.css"
  import { toLatex } from "../math/latex"

  interface Props {
    source?: string
    tex?: string
    display?: boolean
  }
  const { source = "", tex, display = false }: Props = $props()

  const latex = $derived(tex ?? toLatex(source))

  const rendered = $derived.by(() => {
    if (latex === "") return null
    try {
      return katex.renderToString(latex, {
        throwOnError: false,
        displayMode: display,
        strict: "ignore",
      })
    } catch {
      return null
    }
  })
</script>

<svelte:element this={display ? "div" : "span"} class="math" class:display>
  {#if rendered}
    {@html rendered}
  {:else}
    <span class="math-fallback">{source || tex || ""}</span>
  {/if}
</svelte:element>

<style>
  .math {
    color: inherit;
    max-width: 100%;
  }
  .math.display {
    display: block;
    overflow-x: auto;
    overflow-y: hidden;
  }
  .math :global(.katex) {
    color: currentColor;
    font-size: 1.1em;
  }
  .math :global(.katex-error) {
    color: currentColor;
  }
  .math-fallback {
    font-family: var(--mono);
    font-variant-numeric: tabular-nums;
  }
</style>
