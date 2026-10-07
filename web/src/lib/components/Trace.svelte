<script lang="ts">
  import type { TraceNode } from "../engine/schema"

  interface Props {
    nodes: TraceNode[] | undefined
  }
  const { nodes }: Props = $props()

  const byId = $derived(new Map((nodes ?? []).map((n) => [n.id, n])))
  const root = $derived((nodes ?? [])[0])

  function kids(node: TraceNode): TraceNode[] {
    return (node.children ?? []).map((id) => byId.get(id)).filter((n): n is TraceNode => !!n)
  }

  function cited(detail: string | undefined) {
    if (!detail) return null
    const parts = detail.split("·").map((p) => p.trim())
    return parts.length > 1 ? parts : null
  }
</script>

{#snippet branch(node: TraceNode, depth: number)}
  <li>
    <div class="node" data-kind={node.kind}>
      <span class="node-kind">{node.kind}</span>
      <b class="node-label">{node.label}</b>
      {#if node.detail}
        {@const parts = cited(node.detail)}
        {#if parts}
          <span class="node-detail">{parts[0]}</span>
          <span class="node-cite">{parts.slice(1).join(" · ")}</span>
        {:else}
          <span class="node-detail">{node.detail}</span>
        {/if}
      {/if}
    </div>
    {#if kids(node).length > 0 && depth < 7}
      <ul>
        {#each kids(node) as child (child.id)}
          {@render branch(child, depth + 1)}
        {/each}
      </ul>
    {/if}
  </li>
{/snippet}

{#if root}
  <ul class="trace">
    {@render branch(root, 0)}
  </ul>
{:else}
  <p class="empty">No provenance was recorded for this line.</p>
{/if}
