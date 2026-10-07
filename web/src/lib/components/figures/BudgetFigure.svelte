<script lang="ts">
  import { clipName, normalizeShares, shareLabel, type ShareRow } from "../../viz/figures"

  interface Props {
    contributions: ShareRow[]
  }
  const { contributions }: Props = $props()

  const LEFT = 8
  const WIDTH = 324
  const STRIP_Y = 6
  const STRIP_HEIGHT = 20
  const ROWS_Y = 44
  const PITCH = 24
  const TRACK_LEFT = 92
  const TRACK_WIDTH = 192
  const NAME_LIMIT = 10
  const NUMBER_MIN_WIDTH = 16
  const DOMINANT = 0.9

  const ranked = $derived(normalizeShares(contributions))
  const peakSensitivity = $derived(Math.max(0, ...ranked.map((r) => Math.abs(r.sensitivity))))

  const segments = $derived.by(() => {
    let cursor = LEFT
    return ranked.map((row, index) => {
      const width = WIDTH * row.share
      const segment = { x: cursor, width, rank: index + 1, tint: index % 2 === 1 }
      cursor += width
      return segment
    })
  })

  const rows = $derived(
    ranked.map((row, index) => {
      const y = ROWS_Y + index * PITCH
      const sensitivityWidth = peakSensitivity > 0 ? (TRACK_WIDTH * Math.abs(row.sensitivity)) / peakSensitivity : 0
      return {
        y,
        rank: index + 1,
        name: clipName(row.name, NAME_LIMIT),
        full: row.name,
        shareWidth: row.share > 0 ? Math.max(1, TRACK_WIDTH * row.share) : 0,
        sensitivityWidth: row.sensitivity !== 0 ? Math.max(1, sensitivityWidth) : 0,
        percent: shareLabel(row.share),
        detail: `${row.name}: ${shareLabel(row.share)} of the combined uncertainty, sigma ${row.sigma.toPrecision(3)}, sensitivity ${row.sensitivity.toPrecision(3)}`,
      }
    }),
  )

  const lead = $derived(ranked[0])
  const dominated = $derived(ranked.length > 1 && !!lead && lead.share >= DOMINANT)
  const rowsBottom = $derived(ROWS_Y + ranked.length * PITCH)
  const captionY = $derived(rowsBottom + 6)
  const legendY = $derived(rowsBottom + (dominated ? 24 : 8))
  const height = $derived(legendY + 34)

  const summary = $derived(
    ranked.length === 0
      ? "No uncertainty budget."
      : `Uncertainty budget across ${ranked.length} input${ranked.length === 1 ? "" : "s"}: ` +
          ranked.map((r) => `${r.name} ${shareLabel(r.share)}`).join(", ") +
          ".",
  )
</script>

{#if ranked.length > 0}
  <svg viewBox="0 0 340 {height}" role="img" aria-label={summary}>
    <title>uncertainty budget</title>
    <desc>
      {summary} The strip is the whole combined uncertainty cut by share. Each row's thick bar repeats that share and its
      thin line is the input's sensitivity relative to the largest, sign ignored.
    </desc>

    {#each segments as segment (segment.rank)}
      <rect class="segment" class:tint={segment.tint} x={segment.x} y={STRIP_Y} width={segment.width} height={STRIP_HEIGHT} />
      {#if segment.width >= NUMBER_MIN_WIDTH}
        <text class="rank-in-strip" class:on-tint={segment.tint} x={segment.x + segment.width / 2} y={STRIP_Y + 14}
          text-anchor="middle">{segment.rank}</text>
      {/if}
    {/each}
    <text class="note" x={LEFT} y={STRIP_Y + STRIP_HEIGHT + 11}>whole combined uncertainty = 100%</text>

    {#each rows as row (row.rank)}
      <g>
        <title>{row.detail}</title>
        <text class="rank" x={LEFT} y={row.y + 9}>{row.rank}</text>
        <text class="name" x="22" y={row.y + 9}>{row.name}</text>
        <rect class="track" x={TRACK_LEFT} y={row.y} width={TRACK_WIDTH} height="9" />
        {#if row.shareWidth > 0}
          <rect class="share" x={TRACK_LEFT} y={row.y} width={row.shareWidth} height="9" />
        {/if}
        {#if row.sensitivityWidth > 0}
          <rect class="sensitivity" x={TRACK_LEFT} y={row.y + 12} width={row.sensitivityWidth} height="3" />
        {/if}
        <text class="percent" x={LEFT + WIDTH} y={row.y + 9} text-anchor="end">{row.percent}</text>
      </g>
    {/each}

    {#if dominated && lead}
      <text class="caption" x={LEFT} y={captionY + 12}>{clipName(lead.name, 24)} alone supplies {shareLabel(lead.share)}</text>
    {/if}

    <rect class="share" x={LEFT} y={legendY} width="14" height="6" />
    <text class="note" x="28" y={legendY + 6}>share of the combined uncertainty</text>
    <rect class="sensitivity" x={LEFT} y={legendY + 17} width="14" height="3" />
    <text class="note" x="28" y={legendY + 21}>sensitivity, relative to the largest (sign ignored)</text>
  </svg>
{/if}

<style>
  svg {
    display: block;
    width: 100%;
    height: auto;
    overflow: visible;
  }
  .segment {
    fill: var(--series-1);
    stroke: var(--paper);
    stroke-width: 1.2;
  }
  .segment.tint {
    fill: color-mix(in srgb, var(--series-1) 45%, var(--paper));
  }
  .rank-in-strip {
    font-family: var(--mono);
    font-size: 10px;
    fill: var(--paper);
  }
  .rank-in-strip.on-tint {
    fill: var(--ink);
  }
  .rank,
  .percent {
    font-family: var(--mono);
    font-size: 10.5px;
    fill: var(--muted);
  }
  .name {
    font-family: var(--mono);
    font-size: 10.5px;
    fill: var(--ink);
  }
  .note {
    font-family: var(--ui);
    font-size: 9.5px;
    fill: var(--faint);
  }
  .caption {
    font-family: var(--ui);
    font-size: 10.5px;
    fill: var(--ink);
  }
  .track {
    fill: var(--track);
  }
  .share {
    fill: var(--series-1);
  }
  .sensitivity {
    fill: var(--series-2);
  }
</style>
