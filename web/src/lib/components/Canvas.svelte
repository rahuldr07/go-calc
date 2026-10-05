<script lang="ts">
  import type { VizPacket } from "../viz/types"

  interface Props {
    packet: VizPacket
    layer: "model" | "uncertainty" | "evidence"
  }
  const { packet, layer }: Props = $props()

  const two = $derived(packet.inputs.length >= 2)
  const a = $derived(packet.inputs[0])
  const b = $derived(packet.inputs[1])
</script>

<figure class="canvas">
  {#if packet.kind === "measured-rectangle" || packet.kind === "square"}
    <svg viewBox="0 0 340 170" role="img"
      aria-label="{packet.quantity} — a rectangle whose sides are the measured inputs. Explanatory, not to scale.">
      <title>{packet.quantity}</title>
      <desc>{packet.story.meaning}</desc>
      <rect x="54" y="30" width="200" height="104" fill="color-mix(in srgb, var(--series-1) 9%, transparent)"
        stroke="var(--series-1)" stroke-width="1.5" />
      {#if layer === "uncertainty"}
        <rect x="248" y="30" width="12" height="104" fill="var(--series-1)" opacity=".32" />
        <rect x="54" y="128" width="200" height="12" fill="var(--series-2)" opacity=".32" />
      {/if}
      {#if layer === "evidence"}
        <path d="M54 22h200" stroke="var(--muted)" stroke-width="1" stroke-dasharray="3 3" />
        <path d="M46 30v104" stroke="var(--muted)" stroke-width="1" stroke-dasharray="3 3" />
      {/if}
      <text x="154" y="16" text-anchor="middle" font-size="11" fill="var(--muted)" font-family="var(--mono)"
        >{a?.text ?? ""}</text>
      <text x="34" y="86" text-anchor="middle" font-size="11" fill="var(--muted)" font-family="var(--mono)"
        transform="rotate(-90 34 86)">{two ? b.text : (a?.text ?? "")}</text>
      <text x="154" y="88" text-anchor="middle" font-size="14" fill="var(--ink)" font-family="var(--mono)"
        font-weight="500">{packet.quantity}</text>
      {#if packet.kind === "square"}
        <path d="M150 30v6M150 128v6M54 79h6M248 79h6" stroke="var(--series-2)" stroke-width="1.4" />
      {/if}
    </svg>
  {:else if packet.kind === "measurement-interval"}
    <svg viewBox="0 0 340 90" role="img" aria-label="{packet.quantity} — central value with its uncertainty span.">
      <title>{packet.quantity}</title>
      <desc>{packet.story.meaning}</desc>
      <line x1="30" y1="54" x2="310" y2="54" stroke="var(--seam-strong)" stroke-width="1" />
      {#if layer !== "model"}
        <rect x="120" y="44" width="100" height="20" fill="var(--series-1)" opacity=".2" />
        <path d="M120 40v28M220 40v28" stroke="var(--series-1)" stroke-width="1.4" />
      {/if}
      <path d="M170 34v40" stroke="var(--ink)" stroke-width="2" />
      <text x="170" y="26" text-anchor="middle" font-size="13" fill="var(--ink)" font-family="var(--mono)"
        font-weight="500">{packet.quantity}</text>
      {#if layer === "evidence"}
        <text x="120" y="82" text-anchor="middle" font-size="10" fill="var(--faint)" font-family="var(--mono)">−u</text>
        <text x="220" y="82" text-anchor="middle" font-size="10" fill="var(--faint)" font-family="var(--mono)">+u</text>
      {/if}
    </svg>
  {/if}

  <div class="readout">
    <div class="readout"><span class="micro">quantity</span><b>{packet.quantity}</b></div>
    <div class="readout"><span class="micro">dimensions</span><b>{packet.dimensions}</b></div>
    <div class="readout"><span class="micro">source</span><b>{packet.source}</b></div>
  </div>

  <ul class="chain">
    <li><span>observe</span><span>{packet.story.observe}</span></li>
    <li><span>operation</span><span>{packet.story.operation}</span></li>
    <li><span>meaning</span><span>{packet.story.meaning}</span></li>
  </ul>
  <figcaption class="contract-note">geometry is explanatory, not to scale</figcaption>
</figure>
