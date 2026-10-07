<script lang="ts">
  import { digitCount, integerLabel, ratioOf, ratioSentence } from "../../viz/figures"

  interface Props {
    numerator: string
    denominator: string
  }
  const { numerator, denominator }: Props = $props()

  const LEFT = 8
  const WIDTH = 324
  const BAR_HEIGHT = 16
  const SLIVER = 2
  const MAX_CELLS = 12

  const negative = $derived(numerator.startsWith("-"))
  const magnitude = $derived(numerator.replace(/^[+-]/, ""))
  const reading = $derived(ratioOf(magnitude, denominator))

  const numeratorWidth = $derived.by(() => {
    if (!reading || reading.numeratorIsZero) return 0
    const raw = reading.numeratorIsLarger ? WIDTH : WIDTH * reading.fraction
    return Math.max(SLIVER, raw)
  })
  const denominatorWidth = $derived.by(() => {
    if (!reading) return 0
    const raw = reading.numeratorIsLarger ? WIDTH * reading.fraction : WIDTH
    return Math.max(SLIVER, raw)
  })
  const numeratorSliver = $derived(!!reading && !reading.numeratorIsZero && !reading.numeratorIsLarger && WIDTH * reading.fraction < SLIVER)
  const denominatorSliver = $derived(!!reading && reading.numeratorIsLarger && WIDTH * reading.fraction < SLIVER)

  const cellCount = $derived.by(() => {
    if (!reading || reading.numeratorIsLarger) return 0
    if (digitCount(denominator) > 2) return 0
    const count = Number(denominator)
    return count >= 2 && count <= MAX_CELLS ? count : 0
  })
  const cellEdges = $derived(
    Array.from({ length: Math.max(0, cellCount - 1) }, (_, i) => LEFT + (WIDTH * (i + 1)) / cellCount),
  )

  const notes = $derived.by(() => {
    const list: string[] = []
    if (negative) list.push("the value is negative; bars compare sizes only")
    if (numeratorSliver || denominatorSliver) list.push(`the shorter bar is drawn ${SLIVER} units wide so it stays visible`)
    return list
  })

  const numeratorText = $derived((negative ? "−" : "") + integerLabel(magnitude))
  const denominatorText = $derived(integerLabel(denominator))
  const sentence = $derived(reading ? ratioSentence(reading) : "")
  const height = $derived(104 + notes.length * 14)
  const summary = $derived(
    reading
      ? `${numeratorText} over ${denominatorText}: ${sentence}.`
      : `${numerator} over ${denominator} is not a valid ratio.`,
  )
</script>

{#if reading}
  <svg viewBox="0 0 340 {height}" role="img" aria-label={summary}>
    <title>numerator against denominator</title>
    <desc>
      {summary}
      {#if cellCount > 0}The denominator bar is divided into {cellCount} equal parts.{/if}
    </desc>

    <text class="label" x={LEFT} y="12">numerator</text>
    <text class="value" x={LEFT + WIDTH} y="12" text-anchor="end">{numeratorText}</text>
    <rect class="track" x={LEFT} y="18" width={WIDTH} height={BAR_HEIGHT} />
    {#if numeratorWidth > 0}
      <rect class="fill" x={LEFT} y="18" width={numeratorWidth} height={BAR_HEIGHT} />
    {/if}

    <text class="label" x={LEFT} y="56">denominator</text>
    <text class="value" x={LEFT + WIDTH} y="56" text-anchor="end">{denominatorText}</text>
    <rect class="whole" x={LEFT} y="62" width={denominatorWidth} height={BAR_HEIGHT} />

    {#each cellEdges as edge (edge)}
      <path class="cell-seam" d="M{edge} 18v{BAR_HEIGHT}M{edge} 62v{BAR_HEIGHT}" />
    {/each}

    <text class="sentence" x={LEFT} y="102">{sentence}</text>
    {#each notes as note, i (note)}
      <text class="note" x={LEFT} y={116 + i * 14}>{note}</text>
    {/each}
  </svg>
{:else}
  <p class="invalid">Not a valid ratio.</p>
{/if}

<style>
  svg {
    display: block;
    width: 100%;
    height: auto;
    overflow: visible;
  }
  .label,
  .sentence,
  .note {
    font-family: var(--ui);
    font-size: 10px;
    fill: var(--muted);
  }
  .note {
    fill: var(--faint);
  }
  .value {
    font-family: var(--mono);
    font-size: 11px;
    fill: var(--ink);
  }
  .track {
    fill: none;
    stroke: var(--seam);
    stroke-width: 1;
  }
  .fill {
    fill: var(--series-1);
  }
  .whole {
    fill: var(--track);
    stroke: var(--seam-strong);
    stroke-width: 1;
  }
  .cell-seam {
    stroke: var(--paper);
    stroke-width: 1.5;
  }
  .invalid {
    margin: 0;
    font-family: var(--ui);
    font-size: 11px;
    color: var(--faint);
  }
</style>
