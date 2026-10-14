<script lang="ts" module>
  import type { Line as CachedLine } from "../engine/schema"

  const remembered = new Map<string, Map<string, CachedLine>>()
</script>

<script lang="ts">
  import { onMount } from "svelte"
  import type { Line } from "../engine/schema"
  import { evaluateAside, sheet } from "../state/sheet.svelte"
  import { vocabulary } from "../state/vocabulary.svelte"

  interface Props {
    close: () => void
    tryIt: (expression: string) => void
  }
  const { close, tryIt }: Props = $props()

  interface Example {
    input: string
    note?: string
    runnable?: boolean
  }

  interface Topic {
    id: string
    title: string
    intro: string
    examples: Example[]
  }

  const TOPICS: Topic[] = [
    {
      id: "arithmetic",
      title: "Arithmetic",
      intro: "Fractions stay fractions and integers can be any size. Irrational numbers are computed to guaranteed digits.",
      examples: [
        { input: "1/3 + 1/6", note: "held as an exact fraction" },
        { input: "0.1 + 0.2 == 0.3", note: "true here, false in ordinary floating point" },
        { input: "2^100", note: "every digit, exactly" },
        { input: "2^0.5", note: "proven: every digit shown is certain" },
        { input: "18% of 500", note: "percent of" },
        { input: "200 + 10%", note: "add ten percent" },
        { input: "5!", note: "factorial" },
        { input: "17 mod 5", note: "remainder" },
        { input: "0x1F + 0b101", note: "hex and binary literals" },
      ],
    },
    {
      id: "units",
      title: "Units",
      intro: "Write a unit after a number. Convert with in or to. Dimensions are checked, so metres plus seconds is refused.",
      examples: [
        { input: "10 km in mile" },
        { input: "6.8 L/100km * 412 km in L", note: "fuel for a trip" },
        { input: "2 hours + 45 min in min", note: "plurals work" },
        { input: "100 degF in degC", note: "temperature scales" },
        { input: "60 km/h * 30 min", note: "units cancel" },
        { input: "2 KiB in B", note: "data sizes" },
        { input: "1 m + 2 s", note: "refused: length plus time" },
      ],
    },
    {
      id: "measured",
      title: "Measured values",
      intro: "Write ± as +-. The uncertainty is carried through to first order, the standard GUM method. It is an estimate, never presented as a guaranteed bound.",
      examples: [
        { input: "(12.4 +- 0.2 cm) * (7.1 +- 0.1 cm)", note: "area of a measured plate" },
        { input: "expand(9.81 +- 0.02, k=2)", note: "an expanded uncertainty" },
      ],
    },
    {
      id: "compare",
      title: "Comparisons",
      intro: "A comparison is decided only when it can be proven. When the bounds overlap, the answer is undecided, never a guess.",
      examples: [
        { input: "2 > 1" },
        { input: "pi < 22/7", note: "decided from guaranteed bounds" },
        { input: "sqrt(2)*sqrt(2) == 2", note: "undecided rather than wrong" },
        { input: "< > <= >= == !=", note: "also ≤ ≥ ≠", runnable: false },
      ],
    },
    {
      id: "solve",
      title: "Solving and calculus",
      intro: "Exact roots where the algebra allows, proven brackets otherwise. Derivatives are symbolic.",
      examples: [
        { input: "solve(x^2 - 5x + 6 = 0, x)" },
        { input: "solve(x^2 - 2 = 0, x)", note: "irrational roots, proven" },
        { input: "solve(cos(x) = x, x)", note: "no formula, so a proven bracket" },
        { input: "solve(floor(x) = 3, x, 0, 10)", note: "a whole stretch of solutions" },
        { input: "deriv(sin(x^2), x)" },
        { input: "deriv(x^3, x, 2)", note: "the derivative at a point" },
      ],
    },
    {
      id: "lists",
      title: "Lists and statistics",
      intro: "Make a list with brackets or a range. Exact in, exact out. Every item must be the same kind of quantity.",
      examples: [
        { input: "[4, 8, 15, 16, 23, 42][-1]", note: "index from 0, or from the end" },
        { input: "1..10..2", note: "a range with a step" },
        { input: "mean([2, 4, 9])" },
        { input: "stdev([2, 4, 4, 4, 5, 5, 7, 9])", note: "sample; stdevp divides by n" },
        { input: "percentile(1..100, 90)" },
        { input: "sort([2 m, 30 cm, 1 km])", note: "across units of one kind" },
      ],
    },
    {
      id: "whole",
      title: "Whole numbers",
      intro: "Exact at any size. A prime or factor above 3.3 × 10^24 is marked probable, never passed off as proven.",
      examples: [
        { input: "factor(600851475143)" },
        { input: "fib(100)" },
        { input: "gcd(48, 18)" },
        { input: "modpow(2, 1000, 97)" },
        { input: "isprime(2^89 - 1)", note: "probably true, and it says so" },
        { input: "hex(255)" },
      ],
    },
    {
      id: "complex",
      title: "Complex numbers",
      intro: "i is the imaginary unit. Exact wherever the arithmetic stays rational.",
      examples: [
        { input: "sqrt(-4)" },
        { input: "(1+2i)*(3+4i)" },
        { input: "1/(1+i)" },
        { input: "(1+i)^8" },
      ],
    },
    {
      id: "sheet",
      title: "The sheet",
      intro: "Every line is live. Change one and every line that uses it updates. A line starting with # is a heading.",
      examples: [
        { input: "price = 200\nprice * 1.18", note: "name a value and use it" },
        { input: "f(x) = x^2 + 1\nf(3)", note: "define a function" },
        { input: "2 + 3\nans * 2", note: "ans is the answer on the line above" },
      ],
    },
    {
      id: "typing",
      title: "Typing help",
      intro: "Suggestions appear as you type, and a misspelling still finds the name. Tab accepts a suggestion.",
      examples: [
        { input: "sqre(4)", note: "did you mean sqrt?" },
        { input: "sin(pii)", note: "did you mean pi?" },
        { input: "10 km in mils", note: "did you mean mile?" },
      ],
    },
  ]

  const TRUST: [string, string][] = [
    ["exact", "Held as an exact number or fraction. Nothing was approximated anywhere."],
    ["proven", "Bounded by a guaranteed interval. Every digit shown is certain."],
    ["estimate", "Built from measured input, or from a test that is not a proof. It says which."],
    ["held", "A definition, a list or a formula rather than a single number."],
    ["error", "Refused, with the reason and where. go-calc would rather say no than guess."],
  ]

  const KEYS: [string, string][] = [
    ["click an answer", "explain it: the working, the arithmetic and why it can be trusted"],
    ["Tab", "accept a suggestion while typing"],
    ["Ctrl .", "switch between notepad and terminal"],
    ["↑ ↓", "recall earlier lines in the terminal"],
    ["/", "search this page"],
    ["Shift ?", "open this page"],
    ["Esc", "close this page or the explanation"],
  ]

  let query = $state("")
  let search: HTMLInputElement | undefined = $state()
  let body: HTMLElement | undefined = $state()
  let answers: Record<string, Line> = $state({})

  const words = $derived(vocabulary.words)
  const needle = $derived(query.trim().toLowerCase())

  function matches(...texts: (string | undefined)[]) {
    if (needle === "") return true
    return texts.some((t) => t?.toLowerCase().includes(needle))
  }

  const topics = $derived(
    TOPICS.map((t) => ({
      ...t,
      examples: matches(t.title, t.intro) ? t.examples : t.examples.filter((e) => matches(e.input, e.note)),
    })).filter((t) => t.examples.length > 0),
  )

  const reference = $derived(
    words.categories
      .map((category) => ({
        category,
        functions: Object.entries(words.docs)
          .filter(([name, d]) => d.category === category && matches(name, d.signature, d.summary, category))
          .sort(([a], [b]) => a.localeCompare(b)),
      }))
      .filter((group) => group.functions.length > 0),
  )

  const units = $derived(words.units.filter((u) => matches(u, "unit")))
  const constants = $derived(Object.entries(words.constants).filter(([s, m]) => matches(s, m, "constant")))
  const keys = $derived(KEYS.filter(([k, m]) => matches(k, m, "keys")))
  const nothing = $derived(topics.length === 0 && reference.length === 0 && units.length === 0 && constants.length === 0 && keys.length === 0)

  const sections = $derived([
    { id: "start", title: "Start here", shown: needle === "" },
    ...topics.map((t) => ({ id: t.id, title: t.title, shown: true })),
    { id: "functions", title: "All functions", shown: reference.length > 0 },
    { id: "units-list", title: "All units", shown: units.length > 0 },
    { id: "constants", title: "Constants", shown: constants.length > 0 },
    { id: "keys", title: "Keys", shown: keys.length > 0 },
  ].filter((s) => s.shown))

  function linesOf(input: string) {
    return input.split("\n")
  }

  async function compute() {
    const topicInputs = TOPICS.flatMap((t) => t.examples.filter((e) => e.runnable !== false).map((e) => e.input))
    const referenceInputs = Object.values(words.docs).map((d) => d.example)
    const key = `${sheet.digits}|${sheet.angle}`
    const cache = remembered.get(key) ?? new Map<string, Line>()
    remembered.set(key, cache)
    for (const [input, line] of cache) answers[input] = line
    const record = (input: string, line: Line | undefined) => {
      if (!line) return
      cache.set(input, line)
      answers[input] = line
    }
    const run = (inputs: string[]) => {
      const todo = [...new Set(inputs)].filter((input) => !cache.has(input))
      const together = todo.filter(standsAlone)
      const jobs: Promise<void>[] = []
      if (together.length > 0) {
        jobs.push(
          evaluateAside(together).then(
            (result) => together.forEach((input, i) => record(input, result.lines[i])),
            () => undefined,
          ),
        )
      }
      for (const input of todo.filter((input) => !standsAlone(input))) {
        jobs.push(
          evaluateAside(linesOf(input)).then(
            (result) => record(input, result.lines[result.lines.length - 1]),
            () => undefined,
          ),
        )
      }
      return Promise.all(jobs)
    }
    await run(topicInputs)
    await run(referenceInputs)
  }

  function standsAlone(input: string) {
    return !input.includes("\n") && !/\bans\b/.test(input) && !/(^|[^=<>!])=(?!=)/.test(input)
  }

  $effect(() => {
    void words.docs
    void compute()
  })

  function answerText(line: Line | undefined) {
    if (!line) return "…"
    if (line.status === "error") return (line.error ?? "refused").replace(/^\d+:\d+:\s*/, "")
    return line.text
  }

  function jump(id: string) {
    body?.querySelector(`#help-${id}`)?.scrollIntoView({ behavior: "smooth", block: "start" })
  }

  function onKey(event: KeyboardEvent) {
    if (event.key === "Escape") close()
    if (event.key === "/" && document.activeElement !== search) {
      event.preventDefault()
      search?.focus()
    }
  }

  onMount(() => search?.focus())
</script>

<svelte:window onkeydown={onKey} />

{#snippet answer(input: string)}
  {@const line = answers[input]}
  <span class="help-answer" data-status={line?.status ?? "pending"}>
    <span class="help-value">{answerText(line)}</span>
    {#if line}<span class="help-status">{line.status}</span>{/if}
  </span>
{/snippet}

<div class="help-scrim" role="presentation" onclick={close}></div>
<div class="help" role="dialog" aria-modal="true" aria-label="go-calc help">
  <header class="help-head">
    <div class="help-title">
      <h2>Help</h2>
      <p>Every answer is exact, proven, or says why not. Click any example to add it to your sheet.</p>
    </div>
    <label class="help-search">
      <span class="visually-hidden">Search help</span>
      <input bind:this={search} bind:value={query} type="search" placeholder="Search functions, units, examples" autocomplete="off" spellcheck="false" />
      <kbd>/</kbd>
    </label>
    <button type="button" class="chip" onclick={close}>close</button>
  </header>

  <div class="help-layout">
    <nav class="help-nav" aria-label="Help sections">
      {#each sections as s (s.id)}
        <button type="button" onclick={() => jump(s.id)}>{s.title}</button>
      {/each}
    </nav>

    <div class="help-body" bind:this={body}>
      {#if nothing}
        <p class="help-empty">Nothing matches “{query}”. Try a function name such as sqrt, a unit such as km, or a word such as percent.</p>
      {/if}

      {#if needle === ""}
        <section class="help-block" id="help-start">
          <h3>Start here</h3>
          <p class="help-note">
            Type a calculation on any line and its answer appears on the right. Click an answer to see the working,
            the arithmetic inside each step, and why the answer can be trusted. Each answer carries a trust label:
          </p>
          <dl class="help-trust">
            {#each TRUST as [word, meaning] (word)}
              <dt data-status={word}>{word}</dt>
              <dd>{meaning}</dd>
            {/each}
          </dl>
        </section>
      {/if}

      {#each topics as topic (topic.id)}
        <section class="help-block" id={`help-${topic.id}`}>
          <h3>{topic.title}</h3>
          <p class="help-note">{topic.intro}</p>
          <ul class="help-examples">
            {#each topic.examples as ex (ex.input)}
              <li>
                {#if ex.runnable === false}
                  <span class="help-input plain">{ex.input}</span>
                  <span class="help-answer"><span class="help-value muted">{ex.note}</span></span>
                {:else}
                  <button type="button" class="help-input" title="Add to the sheet" onclick={() => tryIt(ex.input)}>
                    {#each linesOf(ex.input) as part}<span>{part}</span>{/each}
                  </button>
                  {@render answer(ex.input)}
                  {#if ex.note}<span class="help-example-note">{ex.note}</span>{/if}
                {/if}
              </li>
            {/each}
          </ul>
        </section>
      {/each}

      {#if reference.length > 0}
        <section class="help-block" id="help-functions">
          <h3>All functions</h3>
          <p class="help-note">{Object.keys(words.docs).length} functions. Each example runs live.</p>
          {#each reference as group (group.category)}
            <h4>{group.category}</h4>
            <ul class="help-functions">
              {#each group.functions as [name, doc] (name)}
                <li>
                  <div class="fn-head">
                    <code class="fn-signature">{doc.signature}</code>
                  </div>
                  <p class="fn-summary">{doc.summary}</p>
                  <div class="fn-example">
                    <button type="button" class="help-input" title="Add to the sheet" onclick={() => tryIt(doc.example)}>{doc.example}</button>
                    {@render answer(doc.example)}
                  </div>
                </li>
              {/each}
            </ul>
          {/each}
        </section>
      {/if}

      {#if units.length > 0}
        <section class="help-block" id="help-units-list">
          <h3>All units</h3>
          <p class="help-note">SI prefixes work on these too: km, mg, µs written us, MW, GiB.</p>
          <ul class="help-chips">
            {#each units as u (u)}<li>{u}</li>{/each}
          </ul>
        </section>
      {/if}

      {#if constants.length > 0}
        <section class="help-block" id="help-constants">
          <h3>Constants</h3>
          <p class="help-note">CODATA 2022 values, from physics.nist.gov/cuu/Constants.</p>
          <dl class="help-defs">
            {#each constants as [symbol, meaning] (symbol)}
              <dt><button type="button" class="help-input" onclick={() => tryIt(symbol)}>{symbol}</button></dt>
              <dd>{meaning}</dd>
            {/each}
          </dl>
        </section>
      {/if}

      {#if keys.length > 0}
        <section class="help-block" id="help-keys">
          <h3>Keys</h3>
          <dl class="help-defs">
            {#each keys as [key, meaning] (key)}
              <dt><kbd>{key}</kbd></dt>
              <dd>{meaning}</dd>
            {/each}
          </dl>
        </section>
      {/if}
    </div>
  </div>
</div>
