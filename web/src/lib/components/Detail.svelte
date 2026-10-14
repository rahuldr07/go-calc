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

  const TRUTHS = ["true", "false", "probably true"]
  const logical = $derived(TRUTHS.includes(line?.text ?? ""))
  const exact = $derived(
    !logical && !line?.text.includes(" = ") && line?.value?.exact && line.value.exact !== line.text ? line.value.exact : "",
  )
  const meaning = $derived(
    line?.status === "estimate" && !line.uncertainty
      ? "Not proven. It rests on a test that is very reliable but is not a proof; the caution under the line says why."
      : MEANING[line?.status ?? "held"],
  )
  const budget = $derived(line?.uncertainty?.budget ?? [])

  function percent(share: number) {
    return `${Math.round(share * 100)}%`
  }

  function agreement(digits: number) {
    if (digits >= 17) return "agrees in every digit float64 has"
    if (digits < 1) return "agrees in no significant digit"
    return `agrees to ${Math.floor(digits)} significant ${Math.floor(digits) === 1 ? "digit" : "digits"}`
  }

  function maths(text: string): string {
    return text
      .replace(/\^2(?![0-9])/g, "²")
      .replace(/\^3(?![0-9])/g, "³")
      .replace(/\bsqrt\(/g, "√(")
      .replace(/\s*\*\s*/g, " × ")
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
      <p class="detail-meaning">{meaning}</p>

      <dl class="detail-facts">
        {#if line.status === "error"}
          <dt>reason</dt><dd class="refused">{located(line.error ?? "refused")}</dd>
        {:else}
          <dt>answer</dt><dd>{line.text}</dd>
        {/if}
        {#if exact}<dt>{line.status === "estimate" ? "midpoint" : "exact"}</dt><dd>{exact}</dd>{/if}
        {#if line.value?.ball && !logical}<dt>bounds</dt><dd>{line.value.ball}</dd>{/if}
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
        <h4>Working <span class="step-count">{working.steps.length} {working.steps.length === 1 ? "step" : "steps"}</span></h4>
        <ol class="timeline">
          {#each working.steps as step (step.n)}
            <li class="timeline-step">
              <span class="timeline-n" aria-hidden="true">{step.n}</span>
              <div class="timeline-body">
                {#if step.explain}<p class="step-explain">{maths(step.explain)}</p>{/if}
                {#if step.work?.length}
                  <div class="step-work" role="group" aria-label="Arithmetic">
                    {#each step.work as line}<p>{maths(line)}</p>{/each}
                  </div>
                {/if}
                <p class="step-result">
                  <code>{maths(step.expr)}</code>
                  <span class="eq">=</span>
                  <b>{maths(step.result)}</b>
                </p>
                {#if step.rule && !step.work?.length && !step.rule.startsWith("unit:") && !step.rule.startsWith("conversion")}
                  <p class="step-rule">{step.rule}</p>
                {/if}
                {#if step.source}<p class="step-rule">{step.source}</p>{/if}
              </div>
            </li>
          {/each}
        </ol>
        <div class="final-answer">
          <span>Answer</span>
          <b>{maths(line.text)}</b>
          <em data-status={line.status}>{line.status}</em>
        </div>
        {#if working.truncated}<p class="detail-aside">{working.note}</p>{/if}
      {:else if working?.note}
        <p class="detail-aside">{working.note}</p>
      {/if}

      {#if working?.summary?.length}
        <div class="trust-note">
          <h4>Why you can trust this</h4>
          {#each working.summary as reason}<p>{reason}</p>{/each}
        </div>
      {/if}
    </div>
  </section>
{/if}
