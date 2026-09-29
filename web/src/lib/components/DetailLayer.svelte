<script lang="ts">
  import { Tabs } from "bits-ui"
  import type { TraceNode } from "../engine/schema"
  import { selectedLine, sheet } from "../state/sheet.svelte"
  import { openLayer, view } from "../state/view.svelte"

  let pane = $state("budget")

  const line = $derived(selectedLine())
  const source = $derived(sheet.source.split("\n")[sheet.selected ?? 0] ?? "")
  const budget = $derived(line?.uncertainty?.budget ?? [])

  function tree(nodes: TraceNode[] | undefined): string {
    if (!nodes || nodes.length === 0) return "no provenance was recorded"
    const byId = new Map(nodes.map((n) => [n.id, n]))
    const out: string[] = []
    const walk = (node: TraceNode, prefix: string, last: boolean, top: boolean) => {
      const stem = top ? "" : `${prefix}${last ? "└─ " : "├─ "}`
      out.push(`${stem}${node.label}${node.detail ? `  ${node.detail}` : ""}`)
      const kids = (node.children ?? []).map((id) => byId.get(id)).filter((n): n is TraceNode => !!n)
      kids.forEach((kid, i) =>
        walk(kid, top ? "" : `${prefix}${last ? "   " : "│  "}`, i === kids.length - 1, false),
      )
    }
    walk(nodes[0], "", true, true)
    return out.join("\n")
  }

  function close() {
    openLayer(false)
  }

  function onKey(event: KeyboardEvent) {
    if (event.key === "Escape") close()
  }
</script>

<svelte:window onkeydown={onKey} />

<aside class="layer" class:on={view.layer} aria-label="Result detail" aria-hidden={!view.layer}>
  {#if line}
    <button class="cls" type="button" onclick={close}>esc ✕</button>
    <div class="lbl">line {line.n + 1} · {source}</div>
    <div class="big">
      {line.status === "error" ? (line.error ?? "refused") : line.text}
    </div>
    <div class="lbl">
      {line.uncertainty ? `gum first-order · ${line.uncertainty.coverage ?? "coverage k = 1"}` : line.status}
    </div>
    <Tabs.Root bind:value={pane}>
      <Tabs.List class="tabs">
        <Tabs.Trigger value="budget">budget</Tabs.Trigger>
        <Tabs.Trigger value="trace">trace</Tabs.Trigger>
        <Tabs.Trigger value="digits">digits</Tabs.Trigger>
        <Tabs.Trigger value="json">json</Tabs.Trigger>
      </Tabs.List>

      <Tabs.Content value="budget" class="pane {pane === 'budget' ? 'on' : ''}">
        {#if budget.length === 0}
          <div class="kv"><span>uncertainty</span><b>none declared — this answer is {line.status}</b></div>
        {:else}
          {#each budget as item (item.name)}
            <div class="bar">
              <span style="color:var(--ivory)">{item.name}</span>
              <span class="t"><i style="width:{(item.share * 100).toFixed(1)}%"></i></span>
              <span style="text-align:right">{(item.share * 100).toFixed(1)} %</span>
            </div>
          {/each}
          <div class="kv">
            {#each budget as item (item.name)}
              <span>u({item.name})</span><b>{item.sigma}</b>
              <span>∂/∂{item.name}</span><b>{item.sensitivity}</b>
            {/each}
            {#if line.uncertainty?.nonlinear}
              <span>linearity</span><b>{line.uncertainty.nonlinear}</b>
            {/if}
          </div>
        {/if}
      </Tabs.Content>

      <Tabs.Content value="trace" class="pane {pane === 'trace' ? 'on' : ''}">
        <div class="tree">{tree(line.trace)}</div>
      </Tabs.Content>

      <Tabs.Content value="digits" class="pane {pane === 'digits' ? 'on' : ''}">
        <div class="kv" style="margin-top:0">
          <span>proven digits</span><b>{line.digits}</b>
          <span>precision cap</span><b>{line.capped ? "reached" : "not reached"}</b>
          {#each line.qualifiers ?? [] as qualifier}
            <span>guarantee</span><b>{qualifier}</b>
          {/each}
          {#if line.stopped}<span>stopped</span><b>{line.stopped}</b>{/if}
        </div>
      </Tabs.Content>

      <Tabs.Content value="json" class="pane {pane === 'json' ? 'on' : ''}">
        <div class="code">{JSON.stringify(line, null, 2)}</div>
      </Tabs.Content>
    </Tabs.Root>
  {/if}
</aside>
