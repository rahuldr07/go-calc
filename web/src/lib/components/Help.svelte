<script lang="ts">
  interface Props {
    close: () => void
  }
  const { close }: Props = $props()

  const SECTIONS: { title: string; note?: string; rows: [string, string][] }[] = [
    {
      title: "Arithmetic",
      note: "Rationals stay exact. Irrationals are computed to proven bounds.",
      rows: [
        ["1/3", "1/3  ≈ 0.333333333333"],
        ["(1+2)*(3+4)", "21"],
        ["2^10000", "a 3011-digit integer, exactly"],
        ["2^0.5", "1.41421356237, proven"],
        ["5!", "120"],
        ["5 mod 3", "2"],
        ["0x1F + 0b101", "36"],
      ],
    },
    {
      title: "Units",
      note: "Convert with `in`. Dimensions are checked, so nonsense is refused.",
      rows: [
        ["9.81 m/s^2 * 2 s", "19.62 (m/s^2)·s"],
        ["10 km in mile", "≈ 6.21371192237 mile"],
        ["1 ft in m", "0.3048 m"],
        ["100 degF in degC", "≈ 37.7777777778 degC"],
        ["2 KiB in B", "2048 B"],
        ["1 year in day", "365.25 day"],
      ],
    },
    {
      title: "Measured values",
      note: "Write ± with +- . Uncertainty propagates to first order; no distribution is assumed.",
      rows: [
        ["(12.4 +- 0.2 cm) * (7.1 +- 0.1 cm)", "0.00880 ± 0.00019 m^2"],
        ["expand(x, k=2)", "widen to a coverage factor"],
      ],
    },
    {
      title: "Comparisons",
      note: "Decided only when the engine can prove it. Overlapping bounds give \"undecided\", never a guess.",
      rows: [
        ["2 > 1", "true"],
        ["1 > 2", "false"],
        ["sqrt(2)*sqrt(2) == 2", "undecided at this precision"],
        ["< > <= >= == !=", "also ≤ ≥ ≠"],
      ],
    },
    {
      title: "Solving",
      note: "Exact roots where the algebra allows, otherwise proven brackets.",
      rows: [
        ["solve(x^2 - 4 = 0, x)", "x = -2,  x = 2"],
        ["solve(x^2 - 2 = 0, x)", "x = -1.41421356237,  x = 1.41421356237"],
        ["solve(x^3 - x = 0, x)", "x = -1,  x = 0,  x = 1"],
        ["solve(cos(x) = x, x)", "0.739085133215"],
        ["solve(eq, x, lo, hi)", "search a range of your own"],
        ["solve(x^2 + 1 = 0, x)", "refused: no real root"],
      ],
    },
    {
      title: "Derivatives",
      note: "Symbolic with two arguments, evaluated at a point with three.",
      rows: [
        ["deriv(x^2, x)", "(2 * x)"],
        ["deriv(sin(x^2), x)", "(cos((x ^ 2)) * (2 * x))"],
        ["deriv(x^2, x, 3)", "6"],
        ["deriv(sqrt(x), x, 4)", "0.25"],
      ],
    },
    {
      title: "Names and functions",
      rows: [
        ["x = 3", "define a value"],
        ["f(x) = x^2", "define a function"],
        ["gcd(12,18)   lcm(4,6)", "6   12"],
        ["root(27,3)", "3"],
        ["isprime(97)", "1"],
        ["floor  ceil  round  abs", "rounding and sign"],
      ],
    },
  ]

  const FUNCTIONS = [
    "sin", "cos", "tan", "asin", "acos", "atan", "atan2",
    "sinh", "cosh", "tanh", "exp", "ln", "log", "log2", "log10",
    "sqrt", "root", "abs", "floor", "ceil", "round", "mod",
    "gcd", "lcm", "isprime", "expand", "solve", "deriv",
  ]

  const CONSTANTS: [string, string][] = [
    ["c", "speed of light in vacuum"],
    ["h", "Planck constant"],
    ["hbar", "reduced Planck constant"],
    ["e_charge", "elementary charge"],
    ["k_B", "Boltzmann constant"],
    ["N_A", "Avogadro constant"],
    ["g0", "standard acceleration of gravity"],
    ["G", "Newtonian constant of gravitation"],
    ["m_e", "electron mass"],
    ["m_p", "proton mass"],
    ["alpha", "fine-structure constant"],
    ["R", "molar gas constant"],
  ]

  const TRUST: [string, string][] = [
    ["exact", "held as an exact number or ratio — no approximation anywhere"],
    ["proven", "bounded by a verified interval; every digit shown is guaranteed"],
    ["estimate", "came from measured input, reported with its uncertainty"],
    ["held", "a definition, not a value"],
    ["error", "refused, with the reason and the column"],
  ]

  function onKey(event: KeyboardEvent) {
    if (event.key === "Escape") close()
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="help-scrim" role="presentation" onclick={close}></div>
<div class="help" role="dialog" aria-modal="true" aria-label="go-calc help">
  <header class="help-head">
    <div>
      <h2>What go-calc can do</h2>
      <p>Every answer is exact or proven. Press <kbd>Esc</kbd> to close.</p>
    </div>
    <button type="button" class="chip" onclick={close}>close</button>
  </header>

  <div class="help-body">
    {#each SECTIONS as section (section.title)}
      <section class="help-block">
        <h3>{section.title}</h3>
        {#if section.note}<p class="help-note">{section.note}</p>{/if}
        <dl class="help-rows">
          {#each section.rows as [input, result] (input)}
            <dt>{input}</dt>
            <dd>{result}</dd>
          {/each}
        </dl>
      </section>
    {/each}

    <section class="help-block">
      <h3>Trust labels</h3>
      <dl class="help-rows wide">
        {#each TRUST as [word, meaning] (word)}
          <dt>{word}</dt>
          <dd>{meaning}</dd>
        {/each}
      </dl>
    </section>

    <section class="help-block">
      <h3>Constants</h3>
      <p class="help-note">CODATA 2022 · physics.nist.gov/cuu/Constants</p>
      <dl class="help-rows wide">
        {#each CONSTANTS as [symbol, name] (symbol)}
          <dt>{symbol}</dt>
          <dd>{name}</dd>
        {/each}
      </dl>
    </section>

    <section class="help-block">
      <h3>Functions</h3>
      <ul class="help-funcs">
        {#each FUNCTIONS as fn (fn)}<li>{fn}</li>{/each}
      </ul>
    </section>

    <section class="help-block">
      <h3>Keys</h3>
      <dl class="help-rows wide">
        <dt>Ctrl .</dt><dd>switch between notepad and terminal</dd>
        <dt>Esc</dt><dd>close this page</dd>
      </dl>
    </section>
  </div>
</div>
