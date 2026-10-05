<script lang="ts">
  import Canvas from "./Canvas.svelte"
  import type { TraceNode } from "../engine/schema"
  import { selectedLine, sheet } from "../state/sheet.svelte"
  import { packetFor } from "../viz/registry"

  type View = "visual" | "why" | "sources" | "origin" | "digits" | "data"

  let chosen = $state<View | null>(null)
  let layer = $state<"model" | "uncertainty" | "evidence">("model")

  const line = $derived(selectedLine())
  const expression = $derived(line ? (sheet.source.split("\n")[line.n] ?? "") : "")
  const budget = $derived(line?.uncertainty?.budget ?? [])
  const packet = $derived(line ? packetFor({ line, expression }) : null)

  const views = $derived(
    (
      [
        ["visual", packet !== null],
        ["why", budget.length > 0],
        ["sources", budget.length > 0],
        ["origin", (line?.trace ?? []).length > 0],
        ["digits", true],
        ["data", true],
      ] as [View, boolean][]
    ).filter(([, ok]) => ok),
  )

  const view = $derived(
    chosen && views.some(([v]) => v === chosen) ? chosen : (views[0]?.[0] ?? "data"),
  )

  $effect(() => {
    void line?.n
    chosen = null
  })

  function tree(nodes: TraceNode[] | undefined): string {
    if (!nodes || nodes.length === 0) return "no provenance recorded"
    const byId = new Map(nodes.map((n) => [n.id, n]))
    const out: string[] = []
    const walk = (node: TraceNode, prefix: string, last: boolean, top: boolean) => {
      out.push(`${top ? "" : prefix + (last ? "└─ " : "├─ ")}${node.label}${node.detail ? "   " + node.detail : ""}`)
      const kids = (node.children ?? []).map((id) => byId.get(id)).filter((n): n is TraceNode => !!n)
      kids.forEach((k, i) => walk(k, top ? "" : prefix + (last ? "   " : "│  "), i === kids.length - 1, false))
    }
    walk(nodes[0], "", true, true)
    return out.join("\n")
  }
</script>

{#if !line}
  <div class="panel">
    <p class="empty">
      <b>No line selected</b>
      Click any answer to see how far to trust it — the uncertainty budget, where each number came from, and how many
      digits are guaranteed.
    </p>
  </div>
{:else}
  <div class="lens-head">
    <div>
      <div class="kicker">{line.status}, line {line.n + 1}</div>
      <h1>{expression}</h1>
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
    <p class="plot-note">Structure now; sampled plots when the engine provides points</p>
  </div>

  <div class="panel">

    {#if view === "visual" && packet}
      <div class="context">
        <span class="tag">selected · line {line.n + 1}</span>
        <span class="expr">{expression}</span>
        <span class="tag">{packet.kind.replace("-", " ")}</span>
      </div>
      <div class="layerbar">
        <div class="pills" role="tablist" aria-label="Canvas layer">
          {#each ["model", "uncertainty", "evidence"] as name (name)}
            <button type="button" role="tab" aria-selected={layer === name}
              onclick={() => (layer = name as typeof layer)}>{name}</button>
          {/each}
        </div>
        <span class="packet" class:contract={packet.source === "contract"}>
          <span class="dot" aria-hidden="true"></span>{packet.source === "contract" ? "contract · no engine packet" : "engine values · UI drawing"}
        </span>
      </div>
      <div class="graph-shell"><div class="graph-core"><Canvas {packet} {layer} /></div></div>
    {:else if view === "why"}
      <h2>What drives the ±</h2>
      <p class="sub">bar length = share of the combined uncertainty</p>
      <p class="empty">
        Each input carries its own uncertainty. The engine propagates them to first order, so a contribution is that
        input's uncertainty multiplied by how strongly the answer responds to it.
      </p>
      <div class="budget">
        {#each budget as item, i (item.name)}
          <div class="budget-row">
            <span class="name">{item.name}</span>
            <span class="meter" class:b={i % 2 === 1}><i style="width:{(item.share * 100).toFixed(1)}%"></i></span>
            <span class="pct">{(item.share * 100).toFixed(1)}%</span>
          </div>
        {/each}
      </div>
    {:else if view === "sources"}
      <h2>Sources of the ±</h2>
      <dl class="facts">
        {#each budget as item (item.name)}
          <dt>u({item.name})</dt><dd>{item.sigma}</dd>
          <dt>∂/∂{item.name}</dt><dd>{item.sensitivity}</dd>
          <dt>share</dt><dd>{(item.share * 100).toFixed(1)}%</dd>
        {/each}
        {#if line.uncertainty?.standard}<dt>combined</dt><dd>{line.uncertainty.standard}</dd>{/if}
        {#if line.uncertainty?.nonlinear}<dt>linearity</dt><dd>{line.uncertainty.nonlinear}</dd>{/if}
      </dl>
    {:else if view === "origin"}
      <h2>Provenance graph</h2>
      <p class="sub">trace nodes and child links</p>
      <pre class="tree">{tree(line.trace)}</pre>
      <p class="honesty">The graph formats trace[]. It does not infer hidden operations.</p>
    {:else if view === "digits"}
      <div class="big-digit">{line.digits}</div>
      <div class="ruler" aria-hidden="true">
        {#each Array(12) as _, i (i)}<i class:on={i < line.digits} data-n={i + 1}></i>{/each}
      </div>
      <dl class="facts" style="margin-top:18px">
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
