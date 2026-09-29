import { readFile } from "node:fs/promises"
import { createRequire } from "node:module"
import { fileURLToPath } from "node:url"
import { beforeAll, describe, expect, it } from "vitest"
import type { Sheet } from "../src/lib/engine/schema"

const here = (path: string) => fileURLToPath(new URL(path, import.meta.url))

let evaluate: (request: string) => string

beforeAll(async () => {
  createRequire(import.meta.url)(here("../public/wasm_exec.js"))
  const Go = (globalThis as unknown as { Go: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): void } }).Go
  const go = new Go()
  const wasm = await WebAssembly.instantiate(await readFile(here("../public/gocalc.wasm")), go.importObject)
  void go.run(wasm.instance)
  evaluate = (globalThis as unknown as { goCalcEval(request: string): string }).goCalcEval
}, 30_000)

const sheetOf = (lines: string[], digits = 12) =>
  JSON.parse(evaluate(JSON.stringify({ lines, digits, angle: "rad" }))) as Sheet

describe("the engine behind the bridge", () => {
  it("answers a whole document in one call", () => {
    const sheet = sheetOf(["h = 12.3 ± 0.2 m", "h * 2"])
    expect(sheet.schema).toBe("go-calc.sheet/1")
    expect(sheet.lines).toHaveLength(2)
    expect(sheet.lines[1].text).toBe("24.60 ± 0.40 m")
    expect(sheet.lines[1].status).toBe("estimate")
  })

  it("keeps every other line when one fails", () => {
    const sheet = sheetOf(["1 + 1", "2 +", "3 * 3"])
    expect(sheet.lines[0].text).toBe("2")
    expect(sheet.lines[1].status).toBe("error")
    expect(sheet.lines[1].error).toContain("unexpected end of input")
    expect(sheet.lines[2].text).toBe("9")
  })

  it("carries the interpretation note and the budget the detail layer needs", () => {
    const sheet = sheetOf(["100 W / 2 m^2"])
    expect(sheet.lines[0].text).toBe("50 W/m^2")
    expect(sheet.lines[0].notes?.[0].text).toContain("read as (100 W)/(2 (m^2))")
    expect(sheet.lines[0].notes?.[0].source).toBe("evaluator")
  })

  it("names the contributions of a measured result", () => {
    const sheet = sheetOf(["h = 12.3 ± 0.2 m", "w = 4.5 ± 0.1 m", "h * w"])
    const budget = sheet.lines[2].uncertainty?.budget ?? []
    expect(budget.map((c) => c.name).sort()).toEqual(["h", "w"])
    expect(budget.reduce((sum, c) => sum + c.share, 0)).toBeCloseTo(1, 6)
  })

  it("carries the provenance tree", () => {
    const sheet = sheetOf(["100 km / 2 h"])
    const trace = sheet.lines[0].trace ?? []
    expect(trace.length).toBeGreaterThan(1)
    expect(trace[0].label).toBe("/")
  })

  it("honours the digit setting", () => {
    expect(sheetOf(["sqrt(2)"]).lines[0].text).toBe("1.41421356237")
    expect(sheetOf(["sqrt(2)"], 20).lines[0].text).toBe("1.4142135623730950488")
  })

  it("holds a provably zero difference to its precision", () => {
    const line = sheetOf(["sqrt(2) - sqrt(2)"]).lines[0]
    expect(line.text).toBe("0")
    expect(line.qualifiers?.[0]).toContain("zero to")
  })
})
