<script lang="ts">
  import type { Derivation, Line } from "../engine/schema"
  import { located } from "../editor/notepad"
  import { lineAt, select, sheet, workingFor } from "../state/sheet.svelte"

  const MEANING: Record<string, string> = {
    exact: "Held as an exact number or ratio. Nothing was approximated anywhere.",
    proven: "Every digit shown is guaranteed: the true value lies inside the bounds below.",
    estimate: "Built from measured input, so it carries an uncertainty. No distribution is assumed.",
    held: "A definition, a list or a symbolic result rather than a single number.",
    error: "Refused. go-calc would rather say no than give an answer it cannot stand behind.",
  }

  const at = $derived(sheet.selected)
  const line = $derived(at === null ? undefined : lineAt(at))
  const source = $derived(at === null ? "" : (sheet.source.split("\n")[at] ?? "").trim())

  let working: Derivation | null = $state(null)

  $effect(() => {
    const current = line
    const input = source
    working = null
    if (!current || current.status === "error" || !current.trace?.length) return
    let live = true
    void workingFor(input, $state.snapshot(current) as Line).then((d) => {
      if (live) working = d
    })
    return () => {
      live = false
    }
  })

  const exact = $derived(line?.value?.exact && line.value.exact !== line.text ? line.value.exact : "")
  const budget = $derived(line?.uncertainty?.budget ?? [])

  function percent(share: number) {
    return `${Math.round(share * 100)}%`
  }

  function agreement(digits: number) {
    if (digits >= 17) return "agrees in every digit float64 has"
    if (digits < 1) return "agrees in no significant digit"
    return `agrees to ${Math.floor(digits)} significant ${Math.floor(digits) === 1 ? "digit" : "digits"}`
  }

  function onKey(event: KeyboardEvent) {
    if (event.key === "Escape" && at !== null && !document.querySelector(".help")) select(null)
  }
</script>

<svelte:window onkeydown={onKey} />

{#if line && at !== null}
  <section class="detail" aria-label={`Line ${at + 1} explained`} data-status={line.status}>
    <header class="detail-head">
      <span class="detail-n">line {at + 1}</span>
      <code class="detail-src">{source}</code>
      <span class="detail-status">{line.status}</span>
      <button type="button" class="chip" onclick={() => select(null)} aria-label="Close the explanation">close</button>
    </header>

    <div class="detail-body">
      <p class="detail-meaning">{MEANING[line.status]}</p>

      <dl class="detail-facts">
        {#if line.status === "error"}
          <dt>reason</dt><dd class="refused">{located(line.error ?? "refused")}</dd>
        {:else}
          <dt>answer</dt><dd>{line.text}</dd>
        {/if}
        {#if exact}<dt>{line.status === "estimate" ? "midpoint" : "exact"}</dt><dd>{exact}</dd>{/if}
        {#if line.value?.ball}<dt>bounds</dt><dd>{line.value.ball}</dd>{/if}
        {#if line.status === "proven"}
          <dt>digits</dt><dd>{line.digits} shown, all guaranteed{line.capped ? " (capped)" : ""}</dd>
        {/if}
        {#if line.float64}
          <dt>float64</dt>
          <dd>
            {line.float64.value}
            <span class="detail-aside">{agreement(line.float64.agrees)}</span>
          </dd>
        {/if}
        {#if line.uncertainty?.coverage}<dt>coverage</dt><dd>{line.uncertainty.coverage}</dd>{/if}
      </dl>

      {#if budget.length > 0}
        <h4>Where the uncertainty comes from</h4>
        <ul class="detail-budget">
          {#each budget as item (item.name)}
            <li>
              <code>{item.name}</code>
              <span class="bar" aria-hidden="true"><span style={`width:${percent(item.share)}`}></span></span>
              <span>{percent(item.share)}</span>
            </li>
          {/each}
        </ul>
      {/if}

      {#if working && working.steps.length > 0}
        <h4>Working</h4>
        <ol class="detail-steps">
          {#each working.steps as step (step.n)}
            <li>
              <code>{step.expr}</code>
              <span class="eq">=</span>
              <b>{step.result}</b>
              {#if step.rule}<span class="detail-aside">{step.rule}</span>{/if}
              {#if step.source}<span class="detail-aside">{step.source}</span>{/if}
            </li>
          {/each}
        </ol>
        {#if working.truncated}<p class="detail-aside">{working.note}</p>{/if}
      {:else if working?.note}
        <p class="detail-aside">{working.note}</p>
      {/if}
    </div>
  </section>
{/if}
