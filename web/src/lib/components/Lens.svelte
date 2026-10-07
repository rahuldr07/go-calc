<script lang="ts">
  import Bracket from "./Bracket.svelte"
  import Canvas from "./Canvas.svelte"
  import MathExpr from "./MathExpr.svelte"
  import Steps from "./Steps.svelte"
  import Trace from "./Trace.svelte"
  import BudgetFigure from "./figures/BudgetFigure.svelte"
  import DimensionFigure from "./figures/DimensionFigure.svelte"
  import MagnitudeFigure from "./figures/MagnitudeFigure.svelte"
  import ScaleFigure from "./figures/ScaleFigure.svelte"
  import { selectedLine, sheet } from "../state/sheet.svelte"
  import { readingOf } from "../viz/bracket"
  import { packetFor } from "../viz/registry"
  import { stepsOf } from "../viz/steps"

  type View = "steps" | "visual" | "scale" | "shape" | "why" | "origin" | "digits" | "data"

  let chosen = $state<View | null>(null)
  let layer = $state<"model" | "uncertainty" | "evidence">("model")

  const line = $derived(selectedLine())
  const expression = $derived(line ? (sheet.source.split("\n")[line.n] ?? "") : "")
  const budget = $derived(line?.uncertainty?.budget ?? [])
  const reading = $derived(readingOf(line))
  const packet = $derived(line ? packetFor({ line, expression }) : null)
  const steps = $derived(stepsOf(line?.trace))
  const ratio = $derived.by(() => {
    const exact = line?.value?.exact
    if (!exact || !exact.includes("/")) return null
    const [numerator, denominator] = exact.split("/")
    return numerator && denominator ? { numerator, denominator } : null
  })

  const views = $derived(
    (
      [
        ["steps", steps.length > 0],
        ["visual", reading !== null],
        ["scale", ratio !== null || (line?.value?.exact ?? "") !== ""],
        ["shape", packet !== null],
        ["why", budget.length > 0],
        ["origin", (line?.trace ?? []).length > 0],
        ["digits", true],
        ["data", true],
      ] as [View, boolean][]
    ).filter(([, ok]) => ok),
  )

  const view = $derived(
    chosen && views.some(([v]) => v === chosen) ? chosen : (views[0]?.[0] ?? "data"),
  )

  const WORD: Record<string, string> = {
    exact: "held exactly",
    proven: "proven to bounds",
    estimate: "measured, with uncertainty",
    held: "held",
    error: "refused",
  }

  $effect(() => {
    void line?.n
    chosen = null
  })
</script>

{#if !line}
  <div class="panel">
    <p class="empty">
      <b>No line selected</b>
      Click any answer to see how far to trust it — the interval it was proved to lie in, what drives the ±, and where
      every number came from.
    </p>
  </div>
{:else}
  <div class="lens-head">
    <div>
      <div class="kicker">{WORD[line.status] ?? line.status} · line {line.n + 1}</div>
      <h1 class="lens-math"><MathExpr source={expression} /></h1>
      <div class="big">{line.status === "error" ? (line.error ?? "refused") : line.text}</div>
    </div>
    <div class="trust-summary">
      <span class="kicker">guaranteed digits</span>
      <b>{line.digits}</b>
      <span class="kicker">{line.capped ? "cap reached" : "cap not reached"}</span>
    </div>
  </div>

  <div class="viewbar">
    <div class="pills" role="tablist">
      {#each views as [name] (name)}
        <button type="button" role="tab" aria-selected={view === name} onclick={() => (chosen = name)}>{name}</button>
      {/each}
    </div>
  </div>

  <div class="panel">
    {#if view === "steps"}
      <h2>Worked step by step</h2>
      <p class="sub">innermost first, exactly as the engine evaluated it</p>
      <Steps {steps} answer={line.text} />
    {:else if view === "visual" && reading}
      <h2>How far this is pinned down</h2>
      <p class="sub">drawn from the engine's own bounds — not a redrawing of the decimal</p>
      <Bracket {reading} />
      {#if line.unit?.dims}
        <h2>Dimensions</h2>
        <DimensionFigure dims={line.unit.dims} />
      {/if}
    {:else if view === "scale"}
      <h2>How big this is</h2>
      <p class="sub">where the answer sits, and what it is a ratio of</p>
      <MagnitudeFigure exact={line.value?.exact} decimal={line.value?.decimal} />
      {#if ratio}
        <ScaleFigure numerator={ratio.numerator} denominator={ratio.denominator} />
      {/if}
    {:else if view === "shape" && packet}
      <h2>What the expression builds</h2>
      <p class="sub">{packet.kind.replace("-", " ")} · geometry is explanatory, not to scale</p>
      <div class="layerbar">
        <div class="pills" role="tablist" aria-label="Canvas layer">
          {#each ["model", "uncertainty", "evidence"] as name (name)}
            <button type="button" role="tab" aria-selected={layer === name}
              onclick={() => (layer = name as typeof layer)}>{name}</button>
          {/each}
        </div>
      </div>
      <div class="graph-shell"><div class="graph-core"><Canvas {packet} {layer} /></div></div>
    {:else if view === "why"}
      <h2>What drives the ±</h2>
      <p class="sub">bar length = share of the combined uncertainty</p>
      <BudgetFigure contributions={budget} />
      <dl class="facts">
        {#each budget as item (item.name)}
          <dt>u({item.name})</dt><dd>{item.sigma}</dd>
          <dt>∂/∂{item.name}</dt><dd>{item.sensitivity}</dd>
        {/each}
        {#if line.uncertainty?.standard !== undefined}
          <dt>combined</dt><dd>{line.uncertainty.standard}</dd>
        {/if}
        {#if line.uncertainty?.coverage}<dt>coverage</dt><dd>{line.uncertainty.coverage}</dd>{/if}
      </dl>
      <p class="honesty">
        Propagated to first order: each contribution is that input's uncertainty times how strongly the answer responds
        to it. No distribution is assumed.
      </p>
    {:else if view === "origin"}
      <h2>Where every number came from</h2>
      <p class="sub">the trace the engine recorded, root first</p>
      <Trace nodes={line.trace} />
      <p class="honesty">This draws trace[] as recorded. It does not infer hidden steps.</p>
    {:else if view === "digits"}
      <h2>Digits you can stand on</h2>
      <div class="big-digit">{line.digits}</div>
      <div class="ruler" aria-hidden="true">
        {#each Array(Math.max(12, Math.min(line.digits, 24))) as _, i (i)}
          <i class:on={i < line.digits} data-n={i + 1}></i>
        {/each}
      </div>
      <dl class="facts">
        <dt>guaranteed</dt><dd>{line.digits}</dd>
        <dt>precision cap</dt><dd>{line.capped ? "reached" : "not reached"}</dd>
        {#each line.qualifiers ?? [] as q}<dt>guarantee</dt><dd>{q}</dd>{/each}
        {#if line.stopped}<dt>stopped</dt><dd>{line.stopped}</dd>{/if}
        {#if line.warning}<dt>float64</dt><dd>{line.warning}</dd>{/if}
      </dl>
    {:else if view === "data"}
      <pre class="raw">{JSON.stringify(line, null, 2)}</pre>
    {/if}
  </div>
{/if}
