# go-calc web front end — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Svelte 5 front end that turns the approved `web/v4.html` mockup into a working calculator driven by the Go/WASM engine.

**Architecture:** One static Vite page, no router. The Go engine is compiled to WASM and runs inside a Web Worker, so a deep precision retry never freezes the page. The whole document crosses the boundary once per keystroke as one `go-calc.sheet/1` request and every line's result comes back together. Svelte holds no engine state: the document text is the only source of truth, and every displayed number comes from the last response. CodeMirror 6 renders the notepad, with each line's value drawn as an end-of-line decoration widget so the value column stays aligned without a second scroll container.

**Tech Stack:** Svelte 5 (runes), Vite 5, TypeScript, CodeMirror 6, Vitest (unit), Playwright (browser). No CSS framework — the mockup's stylesheet is the design system.

**Spec:** `docs/superpowers/specs/2026-09-24-go-calc-design.md` section 4.6, `PRODUCT.md`, `DESIGN.md`, and the approved mockup `web/v4.html`.

## Global Constraints

- **No comments in code.** Names and structure carry the meaning. (`CLAUDE.md`)
- **No AI attribution in commits.** No "Generated with", no `Co-Authored-By` trailer. (`CLAUDE.md`)
- **Commit and push to `main`. No branches.** (`CLAUDE.md`)
- **The front end formats, it never computes.** Principle 6, `PRODUCT.md`. No arithmetic in TypeScript — not rounding a value, not summing a budget, not deciding a digit count. Every number rendered comes from a field of the engine's response. The mockup's demo `ratAdd`/`evalExact`/`ratText` JavaScript engine (`web/v4.html` lines 583-660) is scaffolding and must not be ported.
- **One WASM instance, in a Worker.** Spec section 2 decision 6. The main thread never instantiates the module.
- **`goCalcEval` is the entire engine API.** It takes a JSON string, returns a JSON string. Request: `{"lines": string[], "digits": number, "angle": "rad" | "deg"}`. Response: `go-calc.sheet/1` as defined in `internal/doc/api.go:34-65`.
- **The stateless contract:** the engine keeps nothing between calls. Every request carries the whole document. Never send a diff.
- **Bundle budget:** the WASM module is 1.33 MiB gzipped and must stay under 4 MiB (spec section 2 decision 5). The JS bundle is additional and should stay under 300 KiB gzipped.
- **Theme tokens are copied verbatim** from `web/v4.html` lines 11-31. Do not re-derive, re-name or "improve" a colour. `DESIGN.md` records why each one exists.
- **Accessibility:** WCAG 2.2 AA. Every row value, keycap and tab reachable by keyboard; focus ring is `--signal` at 3px offset; Esc closes the detail layer; ⌘. switches view. (`DESIGN.md` "Rules")
- **Reduced motion:** the dust drift and the layer transition are disabled under `prefers-reduced-motion`. Never animate layout properties; opacity and transform only.

## Review Focus

These are the input classes the spec implies but that no task's happy path exercises. Each one has its test added to the task that owns the code.

1. **The engine returns an error line while the user keeps typing.** An errored line is the normal state of a live notepad, not an exception. The sheet must keep rendering every other line's value, and the error must not clear the previous good value of an unrelated line. — Task 4.
2. **A response arrives out of order.** Keystrokes are faster than a deep precision retry, so response *n* can land after response *n+1*. A stale response must be dropped, never painted. — Task 1.
3. **The WASM module fails to load** (offline, 404, a browser without `WebAssembly.instantiateStreaming`). The page must say so in the status readout rather than sitting silently dead with an editor that does nothing. — Task 1.
4. **A value is far too long for the value column** — 1,000 digits, or an exact rational with a 200-digit numerator. It must truncate or wrap without pushing the expression column off-screen or breaking row alignment. — Task 3.
5. **A line carries several notes and a warning at once.** `100 W / 2 m^2` under a `h = 3` environment produces an interpretation note, a reading note and possibly a float64 warning. They must stack in a stable order under the line, not overlap or collapse into one. — Task 5.

---

## File structure

```
web/
  package.json              scripts: dev, build, preview, test, test:e2e, wasm
  vite.config.ts            svelte plugin, worker format es, public dir
  tsconfig.json             strict
  svelte.config.js          vitePreprocess
  index.html                the app entry; the mockup stays at v4.html untouched
  public/
    gocalc.wasm             produced by wasm/build.sh, gitignored
    wasm_exec.js            produced by wasm/build.sh, gitignored
  src/
    main.ts                 mounts App
    app.css                 the mockup's stylesheet, ported verbatim
    App.svelte              shell: dust, head, stage/term, foot, keypad, layer
    lib/
      engine/
        schema.ts           TypeScript types for go-calc.sheet/1
        worker.ts           Web Worker: loads wasm_exec.js + gocalc.wasm
        client.ts           typed request/response with stale-drop
      state/
        sheet.svelte.ts     runes store: source, response, digits, angle, status
        view.svelte.ts      runes store: notepad|terminal, theme, keypad open
      components/
        Dust.svelte
        Head.svelte
        Foot.svelte
        Notepad.svelte
        Terminal.svelte
        Keypad.svelte
        DetailLayer.svelte
        StatusDots.svelte
        LineNotes.svelte
      editor/
        notepad.ts          CodeMirror state, theme, value decorations
        keys.ts             PAD_GROUPS and CORE key tables
  tests/
    engine.test.ts
    sheet.test.ts
    e2e/notepad.spec.ts
```

---

### Task 1: The engine in a Worker, with a typed client

**Files:**
- Create: `web/package.json`, `web/vite.config.ts`, `web/tsconfig.json`, `web/svelte.config.js`, `web/.gitignore`
- Create: `web/src/lib/engine/schema.ts`, `web/src/lib/engine/worker.ts`, `web/src/lib/engine/client.ts`
- Test: `web/tests/engine.test.ts`
- Modify: `.gitignore` (add `web/node_modules`, `web/dist`, `web/public/gocalc.wasm`, `web/public/wasm_exec.js`)

**Interfaces:**
- Consumes: `goCalcEval(json: string): string`, set on `globalThis` by `wasm/main.go`.
- Produces: `type Sheet`, `type Line`, `type Note`, `type TraceNode`, `type Uncertainty` from `schema.ts`; `class Engine { ready: Promise<void>; evaluate(req: Request): Promise<Sheet>; status: "loading" | "ready" | "failed" }` from `client.ts`.

- [ ] **Step 1: Scaffold the project**

```bash
cd /home/rahul/projects/go-calc/web
npm init -y
npm install --save-dev vite @sveltejs/vite-plugin-svelte svelte typescript svelte-check vitest @vitest/browser playwright @playwright/test
npm install codemirror @codemirror/state @codemirror/view @codemirror/language @codemirror/commands
```

`web/package.json` scripts:

```json
{
  "scripts": {
    "wasm": "sh ../wasm/build.sh",
    "dev": "npm run wasm && vite",
    "build": "npm run wasm && vite build",
    "preview": "vite preview",
    "check": "svelte-check --tsconfig ./tsconfig.json",
    "test": "vitest run",
    "test:e2e": "playwright test"
  }
}
```

- [ ] **Step 2: Write the failing test**

`web/tests/engine.test.ts`:

```ts
import { describe, expect, it } from "vitest"
import { Engine } from "../src/lib/engine/client"

describe("the engine", () => {
  it("answers a whole document in one call", async () => {
    const engine = new Engine()
    await engine.ready
    const sheet = await engine.evaluate({ lines: ["h = 12.3 ± 0.2 m", "h * 2"], digits: 12, angle: "rad" })
    expect(sheet.schema).toBe("go-calc.sheet/1")
    expect(sheet.lines).toHaveLength(2)
    expect(sheet.lines[1].text).toBe("24.60 ± 0.40 m")
    expect(sheet.lines[1].status).toBe("estimate")
  })

  it("drops a response that a later request has overtaken", async () => {
    const engine = new Engine()
    await engine.ready
    const slow = engine.evaluate({ lines: ["2^10000.5"], digits: 900, angle: "rad" })
    const fast = engine.evaluate({ lines: ["1+1"], digits: 12, angle: "rad" })
    await expect(slow).rejects.toThrow("superseded")
    expect((await fast).lines[0].text).toBe("2")
  })

  it("reports a module that will not load", async () => {
    const engine = new Engine({ url: "/no-such.wasm" })
    await expect(engine.ready).rejects.toThrow()
    expect(engine.status).toBe("failed")
  })
})
```

- [ ] **Step 3: Run it to make sure it fails**

Run: `cd web && npx vitest run tests/engine.test.ts`
Expected: FAIL — cannot resolve `../src/lib/engine/client`.

- [ ] **Step 4: Write the schema**

`web/src/lib/engine/schema.ts` — field names and optionality copied from `internal/doc/api.go:34-65`:

```ts
export type Status = "exact" | "proven" | "estimate" | "held" | "error"
export type NoteSource = "parser" | "evaluator"

export interface Note {
  source: NoteSource
  text: string
  column?: number
}

export interface Contribution {
  name: string
  share: number
  sigma: number
  sensitivity: number
}

export interface Uncertainty {
  value: string
  reported: string
  standard?: string
  coverage?: string
  factor?: number
  budget?: Contribution[]
  nonlinear?: string
}

export interface TraceNode {
  id: number
  kind: string
  label: string
  detail?: string
  children?: number[]
}

export interface Line {
  n: number
  text: string
  qualifiers?: string[]
  status: Status
  digits: number
  capped?: boolean
  stopped?: string
  error?: string
  column?: number
  uncertainty?: Uncertainty
  notes?: Note[]
  warning?: string
  trace?: TraceNode[]
}

export interface Sheet {
  schema: "go-calc.sheet/1"
  lines: Line[]
}

export interface Request {
  lines: string[]
  digits: number
  angle: "rad" | "deg"
}
```

- [ ] **Step 5: Write the worker**

`web/src/lib/engine/worker.ts`:

```ts
declare const Go: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): void }
declare function goCalcEval(request: string): string

let started: Promise<void> | null = null

async function start(url: string) {
  importScripts(new URL("/wasm_exec.js", self.location.origin).href)
  const go = new Go()
  const source = await WebAssembly.instantiateStreaming(fetch(url), go.importObject)
  go.run(source.instance)
}

self.onmessage = async (event: MessageEvent) => {
  const { id, kind, url, request } = event.data
  if (kind === "start") {
    started = start(url)
    try {
      await started
      self.postMessage({ id, ok: true })
    } catch (err) {
      self.postMessage({ id, ok: false, error: String(err) })
    }
    return
  }
  try {
    await started
    self.postMessage({ id, ok: true, answer: goCalcEval(request) })
  } catch (err) {
    self.postMessage({ id, ok: false, error: String(err) })
  }
}
```

- [ ] **Step 6: Write the client**

`web/src/lib/engine/client.ts`:

```ts
import type { Request, Sheet } from "./schema"

interface Pending {
  resolve(sheet: Sheet): void
  reject(err: Error): void
}

export class Engine {
  status: "loading" | "ready" | "failed" = "loading"
  readonly ready: Promise<void>
  private readonly worker: Worker
  private readonly pending = new Map<number, Pending>()
  private next = 0
  private latest = 0

  constructor(options: { url?: string } = {}) {
    this.worker = new Worker(new URL("./worker.ts", import.meta.url), { type: "module" })
    this.worker.onmessage = (event) => this.settle(event.data)
    this.ready = this.send("start", options.url ?? "/gocalc.wasm").then(
      () => { this.status = "ready" },
      (err) => { this.status = "failed"; throw err },
    )
  }

  evaluate(request: Request): Promise<Sheet> {
    const id = this.next + 1
    this.latest = id
    return this.send("evaluate", undefined, JSON.stringify(request), id)
  }

  private send(kind: string, url?: string, request?: string, forced?: number): Promise<Sheet> {
    const id = forced ?? ++this.next
    if (forced) this.next = forced
    return new Promise<Sheet>((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      this.worker.postMessage({ id, kind, url, request })
    })
  }

  private settle(data: { id: number; ok: boolean; answer?: string; error?: string }) {
    const waiting = this.pending.get(data.id)
    if (!waiting) return
    this.pending.delete(data.id)
    if (!data.ok) return waiting.reject(new Error(data.error))
    if (data.answer === undefined) return waiting.resolve({ schema: "go-calc.sheet/1", lines: [] })
    if (data.id < this.latest) return waiting.reject(new Error("superseded"))
    waiting.resolve(JSON.parse(data.answer) as Sheet)
  }
}
```

- [ ] **Step 7: Run the tests and make sure they pass**

Run: `cd web && npm run wasm && npx vitest run tests/engine.test.ts`
Expected: PASS, three tests.

- [ ] **Step 8: Commit**

```bash
git add web/package.json web/package-lock.json web/vite.config.ts web/tsconfig.json web/svelte.config.js web/.gitignore web/src/lib/engine web/tests/engine.test.ts .gitignore
git commit -m "feat: the go engine runs in a worker behind a typed client"
```

---

### Task 2: The shell — tokens, dust, head, foot

**Files:**
- Create: `web/index.html`, `web/src/main.ts`, `web/src/app.css`, `web/src/App.svelte`
- Create: `web/src/lib/components/Dust.svelte`, `Head.svelte`, `Foot.svelte`
- Create: `web/src/lib/state/view.svelte.ts`
- Test: `web/tests/e2e/notepad.spec.ts` (first case)

**Interfaces:**
- Consumes: nothing from Task 1 yet.
- Produces: `view` store with `mode: "notepad" | "terminal"`, `theme: "dark" | "light"`, `keypad: boolean`, and `toggleTheme()`, `setMode(m)`, `toggleKeypad()`.

- [ ] **Step 1: Port the stylesheet verbatim**

Copy `web/v4.html` lines 11-230 (everything between `<style>` and `</style>`) into `web/src/app.css` unchanged, then append the two font imports from the mockup's `<head>`. Do not rename a token, change a value, or reformat. `DESIGN.md` is the record of why each one exists.

- [ ] **Step 2: Write the failing test**

`web/tests/e2e/notepad.spec.ts`:

```ts
import { expect, test } from "@playwright/test"

test("the shell renders in both themes", async ({ page }) => {
  await page.goto("/")
  await expect(page.locator(".brand")).toHaveText(/go-calc/)
  await expect(page.locator(".dust i")).toHaveCount(3)
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark")
  await page.getByRole("button", { name: "light" }).click()
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light")
  await page.reload()
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light")
})
```

- [ ] **Step 3: Run it to make sure it fails**

Run: `cd web && npx playwright test tests/e2e/notepad.spec.ts`
Expected: FAIL — no `.brand` element.

- [ ] **Step 4: Write the view store**

`web/src/lib/state/view.svelte.ts`:

```ts
const stored = (key: string) => {
  try { return localStorage.getItem(key) } catch { return null }
}
const remember = (key: string, value: string) => {
  try { localStorage.setItem(key, value) } catch { }
}

const phone = () => matchMedia("(max-width: 860px)").matches

function firstTheme(): "dark" | "light" {
  const saved = stored("gocalc-theme")
  if (saved === "light" || saved === "dark") return saved
  return matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark"
}

export const view = $state({
  mode: "notepad" as "notepad" | "terminal",
  theme: firstTheme(),
  keypad: stored("gocalc-keypad") === "open" || (stored("gocalc-keypad") === null && phone()),
})

export function toggleTheme() {
  view.theme = view.theme === "light" ? "dark" : "light"
  remember("gocalc-theme", view.theme)
}

export function setMode(mode: "notepad" | "terminal") {
  view.mode = mode
}

export function toggleKeypad() {
  view.keypad = !view.keypad
  remember("gocalc-keypad", view.keypad ? "open" : "closed")
}
```

- [ ] **Step 5: Write the shell components**

`web/src/lib/components/Dust.svelte` — the three parallax layers, markup copied from `web/v4.html:242`.

`web/src/lib/components/Head.svelte` — the wordmark, sheet tabs, the Notepad/Terminal/Share keycaps, the digits knob and the amber readout, markup copied from `web/v4.html:246-269`, with the keycaps bound to `setMode` and the `light` keycap to `toggleTheme`.

`web/src/lib/components/Foot.svelte` — engine state, precision, counts, markup from `web/v4.html:410-417`.

`web/src/App.svelte`:

```svelte
<script lang="ts">
  import Dust from "./lib/components/Dust.svelte"
  import Head from "./lib/components/Head.svelte"
  import Foot from "./lib/components/Foot.svelte"
  import { view } from "./lib/state/view.svelte"

  $effect(() => {
    document.documentElement.dataset.theme = view.theme
  })
</script>

<Dust />
<div class="shell">
  <Head />
  <Foot />
</div>
```

`web/src/main.ts`:

```ts
import { mount } from "svelte"
import App from "./App.svelte"
import "./app.css"

mount(App, { target: document.body })
```

- [ ] **Step 6: Run the test and make sure it passes**

Run: `cd web && npx playwright test tests/e2e/notepad.spec.ts`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/index.html web/src/main.ts web/src/app.css web/src/App.svelte web/src/lib/components web/src/lib/state/view.svelte.ts web/tests/e2e
git commit -m "feat: the approved shell in svelte, dark and light"
```

---

### Task 3: The notepad — CodeMirror with value decorations

**Files:**
- Create: `web/src/lib/editor/notepad.ts`, `web/src/lib/components/Notepad.svelte`, `web/src/lib/components/StatusDots.svelte`
- Create: `web/src/lib/state/sheet.svelte.ts`
- Modify: `web/src/App.svelte`
- Test: `web/tests/sheet.test.ts`, `web/tests/e2e/notepad.spec.ts`

**Interfaces:**
- Consumes: `Engine`, `Sheet`, `Line` from Task 1; `view` from Task 2.
- Produces: `sheet` store with `source: string`, `lines: Line[]`, `digits: number`, `angle: "rad" | "deg"`, `engineStatus`, and `setSource(text: string)`; `valueField` (a CodeMirror `StateField<readonly Line[]>`) and `setLines` (a `StateEffect`).

- [ ] **Step 1: Write the failing test**

`web/tests/sheet.test.ts`:

```ts
import { describe, expect, it } from "vitest"
import { sheet, setSource } from "../src/lib/state/sheet.svelte"

describe("the sheet store", () => {
  it("keeps every other line when one fails", async () => {
    await setSource("1 + 1\n2 +\n3 * 3")
    expect(sheet.lines[0].text).toBe("2")
    expect(sheet.lines[1].status).toBe("error")
    expect(sheet.lines[1].error).toContain("unexpected end of input")
    expect(sheet.lines[2].text).toBe("9")
  })

  it("truncates a value too long for the column without losing the expression", async () => {
    await setSource("2^10000")
    const line = sheet.lines[0]
    expect(line.text.length).toBeGreaterThan(3000)
    expect(line.status).toBe("exact")
  })
})
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `cd web && npx vitest run tests/sheet.test.ts`
Expected: FAIL — cannot resolve `sheet.svelte`.

- [ ] **Step 3: Write the sheet store**

`web/src/lib/state/sheet.svelte.ts`:

```ts
import { Engine } from "../engine/client"
import type { Line } from "../engine/schema"

const engine = new Engine()

export const sheet = $state({
  source: "",
  lines: [] as Line[],
  digits: 12,
  angle: "rad" as "rad" | "deg",
  engineStatus: "loading" as "loading" | "ready" | "failed",
  failure: "",
  selected: null as number | null,
  elapsed: 0,
})

engine.ready.then(
  () => { sheet.engineStatus = "ready" },
  (err: Error) => { sheet.engineStatus = "failed"; sheet.failure = err.message },
)

export async function setSource(text: string) {
  sheet.source = text
  if (sheet.engineStatus === "failed") return
  await engine.ready
  try {
    const answer = await engine.evaluate({ lines: text.split("\n"), digits: sheet.digits, angle: sheet.angle })
    sheet.lines = answer.lines
  } catch (err) {
    if ((err as Error).message !== "superseded") throw err
  }
}
```

- [ ] **Step 4: Run the test and make sure it passes**

Run: `cd web && npx vitest run tests/sheet.test.ts`
Expected: PASS, two tests.

- [ ] **Step 5: Write the value decoration**

`web/src/lib/editor/notepad.ts`:

```ts
import { StateEffect, StateField } from "@codemirror/state"
import { Decoration, EditorView, WidgetType, type DecorationSet } from "@codemirror/view"
import type { Line } from "../engine/schema"

export const setLines = StateEffect.define<readonly Line[]>()

class ValueWidget extends WidgetType {
  constructor(readonly line: Line) { super() }

  eq(other: ValueWidget) {
    return other.line.text === this.line.text && other.line.status === this.line.status
  }

  toDOM() {
    const wrap = document.createElement("span")
    wrap.className = "val"
    const num = document.createElement("span")
    num.className = "num"
    num.textContent = this.line.error ? this.line.error : this.line.text
    const st = document.createElement("span")
    st.className = `st ${dotClass(this.line.status)}`
    st.textContent = this.line.status
    wrap.append(num, st)
    return wrap
  }
}

function dotClass(status: Line["status"]) {
  if (status === "proven") return "proven"
  if (status === "estimate") return "est"
  if (status === "held") return "hold"
  if (status === "error") return "err"
  return ""
}

export const valueField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(marks, tr) {
    for (const effect of tr.effects) {
      if (!effect.is(setLines)) continue
      const built = []
      for (const line of effect.value) {
        if (line.n >= tr.state.doc.lines) continue
        const at = tr.state.doc.line(line.n + 1)
        built.push(Decoration.widget({ widget: new ValueWidget(line), side: 1 }).range(at.to))
      }
      return Decoration.set(built)
    }
    return marks.map(tr.changes)
  },
  provide: (field) => EditorView.decorations.from(field),
})
```

- [ ] **Step 6: Write the Notepad component**

`web/src/lib/components/Notepad.svelte` mounts an `EditorView` with `valueField`, a theme extension binding CodeMirror's classes to the mockup's `.row`, `.expr`, `.val`, `.num`, `.st` rules, and an `updateListener` that calls `setSource` on every document change and dispatches `setLines` when `sheet.lines` changes.

Add to `web/src/app.css`, so a long value truncates instead of pushing the expression column:

```css
.cm-line .val{float:right;max-width:46%}
.cm-line .num{display:inline-block;max-width:100%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;vertical-align:bottom}
```

- [ ] **Step 7: Add the browser test**

Append to `web/tests/e2e/notepad.spec.ts`:

```ts
test("a typed line shows its answer on the same row", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("100 km / 2 h")
  await expect(page.locator(".cm-line .num").first()).toContainText("13.8888888889 m/s")
})

test("a very long value does not push the expression off-screen", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("2^10000")
  const expr = page.locator(".cm-line").first()
  await expect(expr).toBeInViewport()
  const box = await expr.boundingBox()
  expect(box!.width).toBeLessThanOrEqual(page.viewportSize()!.width)
})
```

- [ ] **Step 8: Run both suites**

Run: `cd web && npx vitest run && npx playwright test`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add web/src/lib/editor web/src/lib/components/Notepad.svelte web/src/lib/components/StatusDots.svelte web/src/lib/state/sheet.svelte.ts web/src/app.css web/src/App.svelte web/tests
git commit -m "feat: the notepad answers every line as you type"
```

---

### Task 4: Hybrid editing — errors and notes appear when the caret leaves

**Files:**
- Modify: `web/src/lib/editor/notepad.ts`, `web/src/lib/components/Notepad.svelte`
- Test: `web/tests/e2e/notepad.spec.ts`

**Interfaces:**
- Consumes: `valueField`, `setLines` from Task 3.
- Produces: `caretLine` (a `StateField<number>`) exported from `notepad.ts`.

Spec decision 8: the line under the caret recalculates live as it parses; errors, warnings and interpretation notes appear only once the caret leaves the line. An incomplete line shows nothing rather than an error.

- [ ] **Step 1: Write the failing test**

```ts
test("an incomplete line stays quiet until the caret leaves it", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("2 +")
  await expect(page.locator(".cm-line .num")).toHaveCount(0)
  await page.keyboard.press("Enter")
  await expect(page.locator(".cm-line").first()).toContainText("unexpected end of input")
})

test("one failing line does not blank the others", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("1 + 1\n2 +\n3 * 3\n")
  await expect(page.locator(".cm-line").nth(0)).toContainText("2")
  await expect(page.locator(".cm-line").nth(1)).toContainText("unexpected end of input")
  await expect(page.locator(".cm-line").nth(2)).toContainText("9")
})
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `cd web && npx playwright test -g "caret leaves"`
Expected: FAIL — the error shows while typing.

- [ ] **Step 3: Track the caret line**

Add to `web/src/lib/editor/notepad.ts`:

```ts
export const caretLine = StateField.define<number>({
  create: () => 0,
  update(current, tr) {
    if (!tr.selection && !tr.docChanged) return current
    return tr.state.doc.lineAt(tr.state.selection.main.head).number - 1
  },
})
```

- [ ] **Step 4: Withhold the error on the caret's line**

In `ValueWidget.toDOM`, take a second constructor argument `editing: boolean`. When `editing` is true and `line.status === "error"`, render nothing. When `editing` is false, render the error text and the notes. Rebuild the decoration set whenever `caretLine` changes by adding `tr.startState.field(caretLine) !== tr.state.field(caretLine)` to the `update` guard in `valueField`.

- [ ] **Step 5: Run the tests and make sure they pass**

Run: `cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/editor/notepad.ts web/src/lib/components/Notepad.svelte web/tests/e2e/notepad.spec.ts
git commit -m "feat: an incomplete line stays quiet until the caret leaves"
```

---

### Task 5: Notes, warnings and the status dots

**Files:**
- Create: `web/src/lib/components/LineNotes.svelte`
- Modify: `web/src/lib/editor/notepad.ts`, `web/src/app.css`
- Test: `web/tests/e2e/notepad.spec.ts`

**Interfaces:**
- Consumes: `Note` from Task 1, `valueField` from Task 3.
- Produces: nothing new.

`DESIGN.md`: `.st` is status as signal dots — `••••••` proven, `••••○○` estimate, `○○○○○○` held, word-only for exact and input, orange for refused. `.note` is the interpretation line under its row, label in signal caps.

- [ ] **Step 1: Write the failing test**

```ts
test("a line stacks its notes and its warning in a stable order", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("h = 3\n100 W / 2 m^2\n")
  const notes = page.locator(".cm-line").nth(1).locator(".note")
  await expect(notes).toHaveCount(1)
  await expect(notes.first()).toContainText("read as (100 W)/(2 (m^2))")
})

test("status reads as dots, not only as colour", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("sqrt(2)\n")
  await expect(page.locator(".cm-line").first().locator(".st")).toHaveClass(/proven/)
})
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `cd web && npx playwright test -g "stable order"`
Expected: FAIL — no `.note` element.

- [ ] **Step 3: Render notes as a block widget**

Add a `NoteWidget` to `notepad.ts` drawn with `Decoration.widget({ block: true, side: 1 })` at the line end, listing in order: every `note.text` prefixed with its label, then `line.warning`, then each `line.qualifiers` entry. One `.note` div per item so they stack rather than wrap into each other.

- [ ] **Step 4: Run the tests and make sure they pass**

Run: `cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/components/LineNotes.svelte web/src/lib/editor/notepad.ts web/src/app.css web/tests/e2e/notepad.spec.ts
git commit -m "feat: notes, warnings and proof dots under the line"
```

---

### Task 6: The terminal view

**Files:**
- Create: `web/src/lib/components/Terminal.svelte`
- Modify: `web/src/App.svelte`
- Test: `web/tests/e2e/terminal.spec.ts`

**Interfaces:**
- Consumes: `sheet`, `setSource` from Task 3; `view`, `setMode` from Task 2.
- Produces: nothing new.

Spec decision 3 and `PRODUCT.md`: two view modes over one document, switchable at any time, sharing the engine and the document. Switching never loses work.

- [ ] **Step 1: Write the failing test**

```ts
import { expect, test } from "@playwright/test"

test("switching views keeps the work", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("6 * 7")
  await page.getByRole("tab", { name: "terminal" }).click()
  await expect(page.locator(".term .in").last()).toHaveText("6 * 7")
  await expect(page.locator(".term .out").last()).toContainText("42")
  await page.keyboard.press("Meta+.")
  await expect(page.locator(".cm-content")).toContainText("6 * 7")
})
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `cd web && npx playwright test tests/e2e/terminal.spec.ts`
Expected: FAIL — no `.term` element.

- [ ] **Step 3: Write the Terminal component**

Markup from `web/v4.html:377-408`: a scrolling transcript of `.tb` blocks, each `.in` the source line and `.out` the rendered value plus status word, and a `.prompt` input at the foot. Submitting the prompt appends a line to `sheet.source` and calls `setSource`. The transcript is derived from `sheet.source.split("\n")` zipped with `sheet.lines`, so both views read the same document.

- [ ] **Step 4: Run the test and make sure it passes**

Run: `cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/components/Terminal.svelte web/src/App.svelte web/tests/e2e/terminal.spec.ts
git commit -m "feat: the terminal view over the same document"
```

---

### Task 7: The detail layer — budget, trace, digits, JSON

**Files:**
- Create: `web/src/lib/components/DetailLayer.svelte`
- Modify: `web/src/lib/components/Notepad.svelte`, `web/src/App.svelte`
- Test: `web/tests/e2e/detail.spec.ts`

**Interfaces:**
- Consumes: `Line`, `Uncertainty`, `TraceNode` from Task 1.
- Produces: `selected` on the `sheet` store — the index of the inspected line, or `null`.

`DESIGN.md`: the layer slides in from the right with a gradient-to-black edge and a deep shadow, never a bordered panel; a bottom sheet under 860px; Esc closes it.

- [ ] **Step 1: Write the failing test**

```ts
import { expect, test } from "@playwright/test"

test("a value opens its budget, trace, digits and json", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("h = 12.3 ± 0.2 m\nw = 4.5 ± 0.1 m\nh * w\n")
  await page.locator(".cm-line").nth(2).locator(".val").click()
  await expect(page.locator(".layer")).toHaveClass(/on/)
  await expect(page.locator(".layer .bar")).toHaveCount(2)
  await page.getByRole("tab", { name: "trace" }).click()
  await expect(page.locator(".layer .tree")).toContainText("h")
  await page.getByRole("tab", { name: "json" }).click()
  await expect(page.locator(".layer .code")).toContainText("go-calc.sheet/1")
  await page.keyboard.press("Escape")
  await expect(page.locator(".layer")).not.toHaveClass(/on/)
})
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `cd web && npx playwright test tests/e2e/detail.spec.ts`
Expected: FAIL — no `.layer` element.

- [ ] **Step 3: Write the DetailLayer**

Markup from `web/v4.html:436-481`. Four panes:
- **budget** — one `.bar` per `uncertainty.budget` entry, the bar width `share * 100` percent, plus a `.kv` grid of `u(name)`, the sensitivity, the coverage line and `uncertainty.nonlinear`.
- **trace** — the `line.trace` node array rendered as the tree, following each node's `children` ids from the root, each node's `label` and `detail` on one row.
- **digits** — `line.digits`, `line.capped`, `line.stopped` and the `qualifiers`.
- **json** — `JSON.stringify(line, null, 2)`.

Every number here is a field of the response. Compute nothing.

- [ ] **Step 4: Run the test and make sure it passes**

Run: `cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/components/DetailLayer.svelte web/src/lib/components/Notepad.svelte web/src/App.svelte web/tests/e2e/detail.spec.ts
git commit -m "feat: the detail layer over a chosen value"
```

---

### Task 8: The keypad

**Files:**
- Create: `web/src/lib/editor/keys.ts`, `web/src/lib/components/Keypad.svelte`
- Modify: `web/src/App.svelte`
- Test: `web/tests/e2e/keypad.spec.ts`

**Interfaces:**
- Consumes: `view`, `toggleKeypad` from Task 2; the editor instance from Task 3.
- Produces: `PAD_GROUPS` and `CORE` from `keys.ts`.

Spec decision 9: the keypad types into the sheet, so a key press and typed text produce the same line, with the same proof, budget and trace. Keys whose engine is not built render inactive and explain themselves rather than failing; they are never hidden.

- [ ] **Step 1: Port the key tables**

`web/src/lib/editor/keys.ts` — copy `PAD_GROUPS` (`web/v4.html:540-573`) and `CORE` (`web/v4.html:575-582`) verbatim, typed as:

```ts
export type Key = readonly [label: string, insert: string, state: "live" | "held" | "act"]
export const PAD_GROUPS: Record<string, readonly Key[]> = { /* copied */ }
export const CORE: readonly Key[] = [ /* copied */ ]
```

Before committing, reconcile every `"live"` key against the engine: run `gocalc "<insert>1)"` or the equivalent for each and downgrade to `"held"` any that answers `unknown function`. The mockup's marking was a guess; the binary is the truth.

- [ ] **Step 2: Write the failing test**

```ts
import { expect, test } from "@playwright/test"

test("a key press and typed text produce the same line", async ({ page }) => {
  await page.goto("/")
  await page.getByRole("button", { name: "keypad" }).click()
  await page.getByRole("button", { name: "√", exact: true }).click()
  await page.getByRole("button", { name: "2", exact: true }).click()
  await page.getByRole("button", { name: ")", exact: true }).click()
  await expect(page.locator(".cm-content")).toHaveText("sqrt(2)")
  await expect(page.locator(".cm-line .num").first()).toContainText("1.41421356237")
})

test("a key with no engine says so instead of failing", async ({ page }) => {
  await page.goto("/")
  await page.getByRole("button", { name: "keypad" }).click()
  await page.getByRole("tab", { name: "calculus" }).click()
  const key = page.getByRole("button", { name: "∫ dx" })
  await expect(key).toBeDisabled()
  await key.click({ force: true })
  await expect(page.locator(".pad-note")).toContainText("not built yet")
})
```

- [ ] **Step 3: Run it to make sure it fails**

Run: `cd web && npx playwright test tests/e2e/keypad.spec.ts`
Expected: FAIL — no keypad.

- [ ] **Step 4: Write the Keypad**

Markup from `web/v4.html:419-434`: a display strip showing the current entry and `ans`, a group row of tabs swapping the upper block, and the fixed core block. A live key dispatches a CodeMirror insert transaction at the caret. `BACK`, `CLEAR` and `EQ` map to delete-backward, clear-line and newline. A held key renders with the inactive keycap treatment, is `disabled`, and writes its explanation into `.pad-note`.

- [ ] **Step 5: Run the tests and make sure they pass**

Run: `cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/editor/keys.ts web/src/lib/components/Keypad.svelte web/src/App.svelte web/tests/e2e/keypad.spec.ts
git commit -m "feat: the optional keypad types into the sheet"
```

---

### Task 9: The status readout, the digits knob, and the failure path

**Files:**
- Modify: `web/src/lib/components/Head.svelte`, `web/src/lib/components/Foot.svelte`, `web/src/lib/state/sheet.svelte.ts`
- Test: `web/tests/e2e/status.spec.ts`

**Interfaces:**
- Consumes: `sheet.engineStatus`, `sheet.failure`, `sheet.digits`.
- Produces: `setDigits(n: number)` on the sheet store.

- [ ] **Step 1: Write the failing test**

```ts
import { expect, test } from "@playwright/test"

test("the readout says what the engine is doing", async ({ page }) => {
  await page.goto("/")
  await expect(page.locator(".lcd")).toContainText("ready")
})

test("a module that will not load says so rather than sitting dead", async ({ page }) => {
  await page.route("**/gocalc.wasm", (route) => route.fulfill({ status: 404 }))
  await page.goto("/")
  await expect(page.locator(".lcd")).toContainText("engine failed to load")
})

test("the knob changes the proven digits", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("sqrt(2)")
  await expect(page.locator(".cm-line .num").first()).toHaveText("1.41421356237")
  await page.locator(".knob").click()
  await page.keyboard.press("ArrowUp")
  await expect(page.locator(".cm-line .num").first()).toContainText("1.4142135623730950488")
})
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `cd web && npx playwright test tests/e2e/status.spec.ts`
Expected: FAIL — the readout is static text.

- [ ] **Step 3: Wire the readout and the knob**

The `.lcd` shows `sheet.angle`, then `core ready` / `core loading` / `engine failed to load — {sheet.failure}`, then the milliseconds the last `evaluate` took. The knob is a `role="slider"` with `aria-valuenow={sheet.digits}`, responding to arrow keys and to drag, calling `setDigits`, which re-evaluates the whole document.

- [ ] **Step 4: Run the tests and make sure they pass**

Run: `cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/components/Head.svelte web/src/lib/components/Foot.svelte web/src/lib/state/sheet.svelte.ts web/tests/e2e/status.spec.ts
git commit -m "feat: the readout, the digits knob, and an honest failure"
```

---

### Task 10: Dependency lighting (needs an engine change)

**Files:**
- Modify: `internal/doc/api.go` (add `deps` to `lineOut`), `internal/doc/graph.go`
- Modify: `internal/doc/api_test.go`
- Modify: `web/src/lib/engine/schema.ts`, `web/src/lib/editor/notepad.ts`, `web/src/app.css`
- Test: `web/tests/e2e/lighting.spec.ts`

**Interfaces:**
- Consumes: `Line` from Task 1.
- Produces: `Line.deps?: number[]` — the document-line indices that depend on this line, transitively.

`DESIGN.md` "Depth is rank": hovering a value lights it and everything downstream, and pushes the rest back. Spec 4.6 lists dependency lighting as part of the web surface. **The sheet schema does not carry this today** — `lineOut` (`internal/doc/api.go:46-59`) has no dependency field, and the mockup faked it with a hand-written `data-deps` attribute. The `Doc` already holds the graph (`internal/doc/graph.go`), so this is exposing what exists, not computing something new.

- [ ] **Step 1: Write the failing Go test**

Add to `internal/doc/api_test.go`:

```go
func TestTheSheetNamesEachLinesDependents(t *testing.T) {
	req := []byte(`{"lines":["h = 2","w = 3","h * w","1 + 1"],"digits":12}`)
	var out struct {
		Lines []struct {
			N    int   `json:"n"`
			Deps []int `json:"deps"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(EvalRequest(req), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(out.Lines[0].Deps, []int{2}) {
		t.Errorf("line 0 deps = %v, want [2]", out.Lines[0].Deps)
	}
	if !reflect.DeepEqual(out.Lines[1].Deps, []int{2}) {
		t.Errorf("line 1 deps = %v, want [2]", out.Lines[1].Deps)
	}
	if out.Lines[2].Deps != nil || out.Lines[3].Deps != nil {
		t.Errorf("a line nothing depends on carries deps: %v %v", out.Lines[2].Deps, out.Lines[3].Deps)
	}
}
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `go test ./internal/doc/ -run NamesEachLinesDependents`
Expected: FAIL — `out.Lines[0].Deps` is nil.

- [ ] **Step 3: Add the field**

In `internal/doc/api.go`, add `Deps []int \`json:"deps,omitempty"\`` to `lineOut`, and fill it in `(*Doc).sheet` from the existing dependency graph, transitively closed and sorted ascending. Do not add a new traversal if `graph.go` already exposes one — reuse it.

- [ ] **Step 4: Run the Go tests**

Run: `go test ./... && go vet ./... && gofmt -l .`
Expected: PASS, clean.

- [ ] **Step 5: Light the rows**

Add `deps?: number[]` to `Line` in `schema.ts`. In `notepad.ts`, on pointer-enter of a `.val`, add the `lit` class to that line and every line in its `deps`, and `recede` to the rest; clear both on pointer-leave. `.lit` and the 55% opacity for receded rows are already in the ported stylesheet (`DESIGN.md` "Lit rows go to `--lit`; receded rows drop to 55% opacity").

- [ ] **Step 6: Write the browser test**

```ts
import { expect, test } from "@playwright/test"

test("hovering a value lights everything downstream", async ({ page }) => {
  await page.goto("/")
  await page.locator(".cm-content").click()
  await page.keyboard.type("h = 2\nw = 3\nh * w\n1 + 1\n")
  await page.locator(".cm-line").nth(0).locator(".val").hover()
  await expect(page.locator(".cm-line").nth(0)).toHaveClass(/lit/)
  await expect(page.locator(".cm-line").nth(2)).toHaveClass(/lit/)
  await expect(page.locator(".cm-line").nth(3)).toHaveClass(/recede/)
})
```

- [ ] **Step 7: Run both suites**

Run: `go test ./... && cd web && npx playwright test`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/doc/api.go internal/doc/graph.go internal/doc/api_test.go web/src/lib/engine/schema.ts web/src/lib/editor/notepad.ts web/src/app.css web/tests/e2e/lighting.spec.ts
git commit -m "feat: the sheet names each line's dependents, and the notepad lights them"
```

---

### Task 11: Build, CI and deploy

**Files:**
- Modify: `.github/workflows/ci.yml`, `README.md`, `web/vite.config.ts`
- Create: `.github/workflows/pages.yml`

**Interfaces:**
- Consumes: everything.
- Produces: a static `web/dist` and a deployed page.

Spec section 2 decision 4: one repository, and CI builds the WASM module and the front end together so they cannot drift.

- [ ] **Step 1: Add the front end to CI**

Extend `.github/workflows/ci.yml` with a `web` job: set up Node 22, `npm ci`, `npm run wasm`, `npm run check`, `npm test`, `npx playwright install --with-deps chromium`, `npm run test:e2e`, then `npm run build` and assert `web/dist` exists and the gzipped JS bundle is under 300 KiB.

- [ ] **Step 2: Add the Pages workflow**

`.github/workflows/pages.yml` builds on a push to `main` and publishes `web/dist` with `actions/deploy-pages`.

- [ ] **Step 3: Record the measured numbers**

Update `README.md` with the front end's measured bundle sizes beside the engine's existing budget table, and a line saying where the app is hosted.

- [ ] **Step 4: Run the whole suite**

Run:
```bash
cd /home/rahul/projects/go-calc && go test ./... && cd web && npm run check && npm test && npm run build
```
Expected: all green, `web/dist/index.html` exists.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows README.md web/vite.config.ts
git commit -m "ci: build the engine and the front end together, and deploy"
```

---

## Decisions taken for the open questions

The spec's section 3.2 lists these as open. Each is answered here so no task blocks; each is cheap to revisit.

- **First-visit state: an empty sheet, ready to type.** `PRODUCT.md` says the web surface is designed for daily users first, an editor ready to type into, and that demonstrating the product to newcomers must not get in the way. A demo sheet would get in the way. A one-line placeholder in the empty editor points at the ⌘K palette.
- **The ⌘K palette is out of scope for this plan.** It is a discovery surface, and nothing in phase 1 depends on it.
- **A very long value truncates with an ellipsis** and opens in full in the detail layer. Task 3 pins this.
- **`why` does not earn a full surface.** The one-line warning plus the detail layer's budget and digits panes carry it.
- **Settings live on the knob and the keycaps.** There is no settings screen in phase 1.
- **Sharing is out of scope for this plan.** The Share keycap renders and is disabled, with the same "not built yet" treatment as a held keypad key.
