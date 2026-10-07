<script lang="ts">
  import { dimensionFormula, exponentLabel, orderedDimensions } from "../../viz/figures"

  interface Props {
    dims: Record<string, number> | undefined
  }
  const { dims }: Props = $props()

  const BLOCK_CAP = 8
  const BLOCK_MAX = 24
  const PLOT_LEFT = 84
  const PLOT_WIDTH = 248
  const GAP = 1
  const TOP = 6

  interface Block {
    x: number
    width: number
    sign: "above" | "below"
  }
  interface Row {
    symbol: string
    name: string
    label: string
    exponent: number
    y: number
    blocks: Block[]
    capped: boolean
  }

  const terms = $derived(orderedDimensions(dims))
  const formula = $derived(dimensionFormula(terms))
  const reach = (exponent: number): number => Math.min(Math.abs(exponent), BLOCK_CAP)
  const above = $derived(Math.ceil(Math.max(0, ...terms.filter((t) => t.exponent > 0).map((t) => reach(t.exponent)))))
  const below = $derived(Math.ceil(Math.max(0, ...terms.filter((t) => t.exponent < 0).map((t) => reach(t.exponent)))))
  const blockSize = $derived(Math.min(BLOCK_MAX, PLOT_WIDTH / Math.max(1, above + below)))
  const axisX = $derived(PLOT_LEFT + below * blockSize)
  const pitch = $derived(Math.max(blockSize + 8, 28))

  const rows = $derived<Row[]>(
    terms.map((term, index) => {
      const count = reach(term.exponent)
      const positive = term.exponent > 0
      const blocks: Block[] = []
      for (let i = 0; i < Math.ceil(count); i++) {
        const portion = Math.min(1, count - i)
        const width = (blockSize - 2 * GAP) * portion
        const x = positive ? axisX + i * blockSize + GAP : axisX - i * blockSize - GAP - width
        blocks.push({ x, width, sign: positive ? "above" : "below" })
      }
      return {
        symbol: term.symbol,
        name: term.name,
        label: exponentLabel(term.symbol, term.exponent),
        exponent: term.exponent,
        y: TOP + index * pitch,
        blocks,
        capped: Math.abs(term.exponent) > BLOCK_CAP,
      }
    }),
  )

  const capped = $derived(rows.some((r) => r.capped))
  const plotBottom = $derived(TOP + terms.length * pitch - (pitch - blockSize))
  const footerY = $derived(plotBottom + 22)
  const hasAbove = $derived(terms.some((t) => t.exponent > 0))
  const hasBelow = $derived(terms.some((t) => t.exponent < 0))
  const keyY = $derived(footerY + 18 + (capped ? 14 : 0))
  const height = $derived(keyY + 12)

  const summary = $derived(
    terms.length === 0
      ? "Dimensionless: every base dimension cancels."
      : `Dimensions ${formula}. ` + terms.map((t) => `${t.name} to the power ${t.exponent}`).join(", ") + ".",
  )
</script>

{#if terms.length === 0}
  <svg viewBox="0 0 340 40" role="img" aria-label={summary}>
    <title>dimensionless</title>
    <desc>{summary}</desc>
    <text class="plain" x="8" y="24">dimensionless: every base dimension cancels</text>
  </svg>
{:else}
  <svg viewBox="0 0 340 {height}" role="img" aria-label={summary}>
    <title>dimensions {formula}</title>
    <desc>{summary} Each block is one power of that base dimension: blocks right of the line multiply, blocks left of it divide.</desc>

    <path class="axis" d="M{axisX} {TOP - 2}V{plotBottom + 2}" />

    {#each rows as row (row.symbol)}
      <text class="exponent" x="8" y={row.y + blockSize / 2 + 1}>{row.label}</text>
      <text class="name" x="8" y={row.y + blockSize / 2 + 12}>{row.name}</text>
      {#each row.blocks as block, i (i)}
        <rect class={block.sign} x={block.x} y={row.y} width={block.width} height={blockSize} />
      {/each}
      {#if row.capped}
        <text class="more" x={row.exponent > 0 ? axisX + BLOCK_CAP * blockSize + 3 : axisX - BLOCK_CAP * blockSize - 3}
          y={row.y + blockSize / 2 + 3} text-anchor={row.exponent > 0 ? "start" : "end"}>⋯</text>
      {/if}
    {/each}

    <text class="formula" x="8" y={footerY}>{formula}</text>
    {#if capped}
      <text class="note" x="8" y={footerY + 14}>blocks stop at {BLOCK_CAP}; read the exponent for larger powers</text>
    {/if}

    {#if hasAbove}
      <rect class="above" x="8" y={keyY - 8} width="8" height="8" />
      <text class="note" x="20" y={keyY}>multiplies</text>
    {/if}
    {#if hasBelow}
      <rect class="below" x="86" y={keyY - 8} width="8" height="8" />
      <text class="note" x="98" y={keyY}>divides</text>
    {/if}
    <text class="note" x="332" y={keyY} text-anchor="end">one block = one power</text>
  </svg>
{/if}

<style>
  svg {
    display: block;
    width: 100%;
    height: auto;
    overflow: visible;
  }
  .exponent {
    font-family: var(--mono);
    font-size: 14px;
    font-weight: 600;
    fill: var(--ink);
  }
  .name,
  .note,
  .plain,
  .more {
    font-family: var(--ui);
    font-size: 9.5px;
    fill: var(--faint);
  }
  .plain {
    font-size: 11px;
    fill: var(--muted);
  }
  .more {
    font-size: 12px;
    fill: var(--muted);
  }
  .formula {
    font-family: var(--mono);
    font-size: 12px;
    fill: var(--ink);
  }
  .axis {
    stroke: var(--seam-strong);
    stroke-width: 1;
  }
  .above {
    fill: var(--series-1);
  }
  .below {
    fill: var(--series-2);
  }
</style>
