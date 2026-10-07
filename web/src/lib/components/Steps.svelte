<script lang="ts">
  import MathExpr from "./MathExpr.svelte"
  import type { Step } from "../viz/steps"

  interface Props {
    steps: Step[]
    answer: string
  }
  const { steps, answer }: Props = $props()

  const cited = $derived.by(() => {
    const lines: string[] = []
    for (const step of steps) {
      if (!step.note) continue
      for (const raw of step.note.split("\n")) {
        const text = raw.trim()
        if (text !== "" && !lines.includes(text)) lines.push(text)
      }
    }
    return lines
  })
</script>

{#if steps.length === 0}
  <p class="empty">This line resolves in a single step — there is no working to show.</p>
{:else}
  <ol class="steps">
    {#each steps as step (step.id)}
      <li class="step" style="--depth:{Math.min(step.depth, 5)}">
        <span class="step-n">{step.n}</span>
        <div class="step-body">
          <div class="step-math"><MathExpr source={step.expression} /></div>
          {#if step.value}
            <div class="step-value"><span class="micro">gives</span><b>{step.value}</b></div>
          {/if}
          {#if step.inputs.length > 0}
            <div class="step-inputs">
              {#each step.inputs as input (input.label)}
                <span class="chiplet">
                  <b>{input.label}</b>{#if input.value && input.value !== input.label}<em>=</em><i
                      >{input.value}</i
                    >{/if}
                </span>
              {/each}
            </div>
          {/if}
        </div>
      </li>
    {/each}
    <li class="step final">
      <span class="step-n">=</span>
      <div class="step-body">
        <div class="step-value final-value"><b>{answer}</b></div>
      </div>
    </li>
  </ol>

  {#if cited.length > 0}
    <ul class="cites">
      {#each cited as text (text)}
        <li>{text}</li>
      {/each}
    </ul>
  {/if}
{/if}
