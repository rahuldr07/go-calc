<script lang="ts">
  import {
    headlineFor,
    logDomain,
    magnitudeOf,
    niceTicks,
    placeOn,
    scientificLabel,
    tickLabel,
  } from "../../viz/figures"

  interface Props {
    exact?: string
    decimal?: string
  }
  const { exact, decimal }: Props = $props()

  const AXIS_LEFT = 16
  const AXIS_RIGHT = 324
  const AXIS_Y = 66
  const MINIMUM_SPAN = 4
  const TICK_TARGET = 5
  const COUNT_MODE_DIGITS = 15

  const magnitude = $derived(magnitudeOf({ exact, decimal }))
  const placeable = $derived(!!magnitude && magnitude.sign !== 0 && Number.isFinite(magnitude.log10))
  const countsDigits = $derived(
    !!magnitude && magnitude.isInteger && magnitude.wholeDigits !== null && magnitude.wholeDigits > COUNT_MODE_DIGITS,
  )

  const domain = $derived(placeable && magnitude ? logDomain(magnitude.log10, MINIMUM_SPAN) : ([0, 1] as [number, number]))
  const ticks = $derived(niceTicks(domain[0], domain[1], TICK_TARGET))
  const markerX = $derived(
    placeable && magnitude ? placeOn(magnitude.log10, domain[0], domain[1], AXIS_LEFT, AXIS_RIGHT) : AXIS_LEFT,
  )
  const anchor = $derived(markerX < 90 ? "start" : markerX > 250 ? "end" : "middle")

  const headline = $derived(magnitude ? headlineFor(magnitude) : "")
  const subline = $derived.by(() => {
    if (!magnitude || !placeable) return ""
    if (countsDigits) return `≈ ${scientificLabel(magnitude.log10)}`
    const decades = Math.abs(magnitude.log10)
    if (decades < 0.05) return "about 1"
    return `${decades.toFixed(1)} decades ${magnitude.log10 < 0 ? "below" : "above"} 1`
  })
  const notes = $derived.by(() => {
    const list = ["log scale: each step multiplies by ten"]
    if (magnitude && magnitude.sign < 0) list.push("negative value, placed by its size")
    return list
  })
  const height = $derived(98 + notes.length * 13)

  const summary = $derived(
    !magnitude
      ? "No magnitude available."
      : !placeable
        ? "The value is zero, which has no position on a log scale."
        : `${headline}${subline && !countsDigits ? `, ${subline}` : ""}. Placed on a log scale.`,
  )
</script>

{#if magnitude}
  {#if placeable}
    <svg viewBox="0 0 340 {height}" role="img" aria-label={summary}>
      <title>magnitude on a log scale</title>
      <desc>
        {summary}
        {#if countsDigits}The digits are not drawn; only their count is.{/if}
        The axis runs from 10^{domain[0]} to 10^{domain[1]}.
      </desc>

      <text class="headline" x={markerX} y="18" text-anchor={anchor}>{headline}</text>
      <text class="subline" x={markerX} y="33" text-anchor={anchor}>{subline}</text>

      <path class="stem" d="M{markerX} 40V{AXIS_Y}" />
      <path class="axis" d="M{AXIS_LEFT} {AXIS_Y}H{AXIS_RIGHT}" />
      <circle class="dot" cx={markerX} cy={AXIS_Y} r="3.5" />

      {#each ticks as tick (tick.value)}
        {@const x = placeOn(tick.value, domain[0], domain[1], AXIS_LEFT, AXIS_RIGHT)}
        <path class={tick.isReference ? "tick reference" : "tick"} d="M{x} {AXIS_Y}v{tick.isReference ? 8 : 5}" />
        <text class={tick.isReference ? "tick-label reference" : "tick-label"} x={x} y={AXIS_Y + 20} text-anchor="middle"
          >{tickLabel(tick.value)}</text>
      {/each}

      {#each notes as note, i (note)}
        <text class="note" x="8" y={AXIS_Y + 40 + i * 13}>{note}</text>
      {/each}
    </svg>
  {:else}
    <svg viewBox="0 0 340 40" role="img" aria-label={summary}>
      <title>zero</title>
      <desc>{summary}</desc>
      <text class="note" x="8" y="24">zero has no place on a log scale</text>
    </svg>
  {/if}
{/if}

<style>
  svg {
    display: block;
    width: 100%;
    height: auto;
    overflow: visible;
  }
  .headline {
    font-family: var(--mono);
    font-size: 12.5px;
    font-weight: 600;
    fill: var(--ink);
  }
  .subline,
  .note {
    font-family: var(--ui);
    font-size: 9.5px;
    fill: var(--faint);
  }
  .subline {
    fill: var(--muted);
  }
  .stem {
    stroke: var(--ink);
    stroke-width: 1.8;
  }
  .axis {
    stroke: var(--seam-strong);
    stroke-width: 1;
  }
  .dot {
    fill: var(--ink);
  }
  .tick {
    stroke: var(--seam-strong);
    stroke-width: 1;
  }
  .tick.reference {
    stroke: var(--muted);
    stroke-width: 1.5;
  }
  .tick-label {
    font-family: var(--mono);
    font-size: 9.5px;
    fill: var(--faint);
  }
  .tick-label.reference {
    fill: var(--ink);
  }
</style>
