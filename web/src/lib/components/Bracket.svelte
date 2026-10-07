<script lang="ts">
  import type { Reading } from "../viz/bracket"

  interface Props {
    reading: Reading
  }
  const { reading }: Props = $props()

  const span = $derived(reading.kind === "proven" && reading.lo && reading.hi)
</script>

<figure class="bracket">
  <div class="digits" aria-label="the digits this answer is pinned to">
    <span class="certain">{reading.certain}</span>
    {#if span && (reading.loTail || reading.hiTail)}
      <span class="fork" aria-label="where the two bounds diverge">
        <i>{reading.loTail || "·"}</i>
        <i>{reading.hiTail || "·"}</i>
      </span>
    {/if}
    {#if reading.unit}<span class="unit">{reading.unit}</span>{/if}
  </div>
  <div class="digit-key">
    {#if span}
      <span><i class="swatch certain"></i>both bounds agree this far</span>
      <span><i class="swatch divergent"></i>and only here do they differ</span>
    {:else if reading.kind === "exact" && reading.exactIsWhole}
      <span><i class="swatch certain"></i>every digit is exact</span>
    {:else if reading.kind === "exact"}
      <span><i class="swatch certain"></i>exact as a ratio — the decimal is cut here, not rounded away</span>
    {:else if reading.kind === "measured"}
      <span><i class="swatch certain"></i>digits the reported uncertainty supports</span>
    {/if}
  </div>

  {#if span}
    <svg viewBox="0 0 340 96" role="img"
      aria-label="The proven interval, magnified. The answer lies somewhere inside it.">
      <title>proven interval</title>
      <desc>Lower bound {reading.lo}, upper bound {reading.hi}. Magnified; the real width is {reading.width}.</desc>
      <rect x="40" y="38" width="260" height="18" fill="color-mix(in srgb, var(--series-2) 22%, transparent)" />
      <path d="M40 28v38M300 28v38" stroke="var(--series-2)" stroke-width="1.8" />
      <text x="40" y="22" font-size="10" fill="var(--muted)" font-family="var(--mono)">{reading.lo}</text>
      <text x="300" y="22" text-anchor="end" font-size="10" fill="var(--muted)" font-family="var(--mono)"
        >{reading.hi}</text>
      <text x="170" y="52" text-anchor="middle" font-size="10.5" fill="var(--ink)" font-family="var(--ui)"
        >somewhere in here</text>
      <text x="170" y="82" text-anchor="middle" font-size="10" fill="var(--faint)" font-family="var(--ui)"
        >magnified · true width {reading.width}</text>
    </svg>
    <dl class="facts">
      <dt>shown</dt>
      <dd>{reading.shown}</dd>
    </dl>
    <p class="claim">
      The engine proved the answer lies inside this interval — it does not claim to know where. The digits above are the
      ones both ends agree on, so no further computation can change them.
    </p>
  {:else if reading.kind === "exact"}
    <svg viewBox="0 0 340 96" role="img" aria-label="An exact value: the interval has no width.">
      <title>exact value</title>
      <desc>{reading.exact} is exact. There is no interval to narrow.</desc>
      <line x1="40" y1="47" x2="300" y2="47" stroke="var(--seam-strong)" stroke-width="1" />
      <path d="M170 31v32" stroke="var(--ink)" stroke-width="2.5" />
      <circle cx="170" cy="47" r="4" fill="var(--ink)" />
      <text x="170" y="80" text-anchor="middle" font-size="10" fill="var(--faint)" font-family="var(--ui)"
        >interval width 0</text>
    </svg>
    <dl class="facts">
      <dt>exact</dt>
      <dd class="exact-value">{reading.exact}</dd>
      <dt>shown</dt>
      <dd>{reading.shown}</dd>
    </dl>
    <p class="claim">
      {reading.exactIsWhole
        ? "This answer is held as an exact number, not an approximation of one."
        : "This answer is held as an exact ratio. The decimal above is a rendering of it, cut to the digits you asked for."}
    </p>
  {:else if reading.kind === "measured"}
    <svg viewBox="0 0 340 96" role="img" aria-label="A measured value with its standard uncertainty.">
      <title>measured value</title>
      <desc>{reading.shown} with reported uncertainty {reading.reported ?? "unknown"}.</desc>
      <line x1="30" y1="47" x2="310" y2="47" stroke="var(--seam-strong)" stroke-width="1" />
      <rect x="110" y="37" width="120" height="20" fill="color-mix(in srgb, var(--series-1) 18%, transparent)" />
      <path d="M110 31v32M230 31v32" stroke="var(--series-1)" stroke-width="1.5" />
      <path d="M170 27v40" stroke="var(--ink)" stroke-width="2" />
      <text x="110" y="80" text-anchor="middle" font-size="10" fill="var(--faint)" font-family="var(--mono)">−u</text>
      <text x="230" y="80" text-anchor="middle" font-size="10" fill="var(--faint)" font-family="var(--mono)">+u</text>
    </svg>
    <dl class="facts">
      <dt>reported</dt>
      <dd>± {reading.reported ?? "—"}</dd>
      {#if reading.sigma !== undefined}<dt>standard</dt><dd>{reading.sigma}</dd>{/if}
    </dl>
    <p class="claim">
      This came from measured inputs, so the engine reports a standard uncertainty rather than a proof. No distribution
      is assumed.
    </p>
  {/if}

  {#if reading.dims}
    <div class="dim-row">
      <span class="micro">dimensions</span>
      <b>{reading.dims}</b>
      {#if reading.unit}<span class="micro">as written</span><b>{reading.unit}</b>{/if}
    </div>
  {/if}
</figure>
