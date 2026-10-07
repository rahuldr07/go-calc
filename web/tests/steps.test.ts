import { describe, expect, it } from "vitest"
import type { TraceNode } from "../src/lib/engine/schema"
import { stepsOf } from "../src/lib/viz/steps"

const CODATA = "CODATA 2022 · physics.nist.gov/cuu/Constants"

const massEnergy: TraceNode[] = [
  { id: 23, kind: "operation", label: "*", children: [19, 22] },
  { id: 19, kind: "constant", label: "m_e", detail: `electron mass · ${CODATA}` },
  { id: 22, kind: "operation", label: "^", children: [20, 21] },
  { id: 20, kind: "constant", label: "c", detail: `speed of light in vacuum · ${CODATA}` },
  { id: 21, kind: "literal", label: "2" },
]

const sumsProduct: TraceNode[] = [
  { id: 1, kind: "operation", label: "*", children: [2, 5], value: "21" },
  { id: 2, kind: "operation", label: "+", children: [3, 4], value: "3" },
  { id: 3, kind: "literal", label: "1", value: "1" },
  { id: 4, kind: "literal", label: "2", value: "2" },
  { id: 5, kind: "operation", label: "+", children: [6, 7], value: "7" },
  { id: 6, kind: "literal", label: "3", value: "3" },
  { id: 7, kind: "literal", label: "4", value: "4" },
]

function binary(id: number, label: string, left: number, right: number): TraceNode {
  return { id, kind: "operation", label, children: [left, right] }
}

function literal(id: number, label: string): TraceNode {
  return { id, kind: "literal", label }
}

function expressionOf(nodes: TraceNode[]): string {
  const steps = stepsOf(nodes)
  return steps[steps.length - 1].expression
}

describe("stepsOf", () => {
  it("returns nothing for undefined, empty, and a lone literal", () => {
    expect(stepsOf(undefined)).toEqual([])
    expect(stepsOf([])).toEqual([])
    expect(stepsOf([literal(1, "7")])).toEqual([])
  })

  it("orders m_e * c^2 bottom-up with citations on the consuming step", () => {
    const steps = stepsOf(massEnergy)
    expect(steps.map((step) => step.id)).toEqual([22, 23])
    expect(steps.map((step) => step.n)).toEqual([1, 2])
    expect(steps.map((step) => step.expression)).toEqual(["c^2", "m_e * c^2"])
    expect(steps.map((step) => step.depth)).toEqual([1, 0])
    expect(steps[0].inputs).toEqual([{ label: "c" }, { label: "2" }])
    expect(steps[0].note).toBe(`c: speed of light in vacuum · ${CODATA}`)
    expect(steps[1].inputs).toEqual([{ label: "m_e" }, { label: "c^2" }])
    expect(steps[1].note).toBe(`m_e: electron mass · ${CODATA}`)
  })

  it("works (1+2)*(3+4) innermost first and carries values", () => {
    const steps = stepsOf(sumsProduct)
    expect(steps.map((step) => step.expression)).toEqual(["1 + 2", "3 + 4", "(1 + 2) * (3 + 4)"])
    expect(steps.map((step) => step.value)).toEqual(["3", "7", "21"])
    expect(steps[2].inputs).toEqual([
      { label: "1 + 2", value: "3" },
      { label: "3 + 4", value: "7" },
    ])
    expect(steps.map((step) => step.depth)).toEqual([1, 1, 0])
  })

  it("parenthesises minimally", () => {
    const product = [
      binary(1, "+", 2, 3),
      literal(2, "1"),
      binary(3, "*", 4, 5),
      literal(4, "2"),
      literal(5, "3"),
    ]
    expect(expressionOf(product)).toBe("1 + 2 * 3")

    const subtractionOnRight = [binary(1, "-", 2, 3), literal(2, "1"), binary(3, "-", 4, 5), literal(4, "2"), literal(5, "3")]
    expect(expressionOf(subtractionOnRight)).toBe("1 - (2 - 3)")

    const subtractionOnLeft = [binary(1, "-", 2, 3), binary(2, "-", 4, 5), literal(3, "3"), literal(4, "1"), literal(5, "2")]
    expect(expressionOf(subtractionOnLeft)).toBe("1 - 2 - 3")

    const powerTower = [binary(1, "^", 2, 3), literal(2, "2"), binary(3, "^", 4, 5), literal(4, "3"), literal(5, "2")]
    expect(expressionOf(powerTower)).toBe("2^3^2")

    const powerOfPower = [binary(1, "^", 2, 3), binary(2, "^", 4, 5), literal(3, "2"), literal(4, "2"), literal(5, "3")]
    expect(expressionOf(powerOfPower)).toBe("(2^3)^2")

    const sumPower = [binary(1, "^", 2, 3), binary(2, "+", 4, 5), literal(3, "2"), literal(4, "1"), literal(5, "2")]
    expect(expressionOf(sumPower)).toBe("(1 + 2)^2")

    const divisionByProduct = [binary(1, "/", 2, 3), literal(2, "1"), binary(3, "*", 4, 5), literal(4, "2"), literal(5, "3")]
    expect(expressionOf(divisionByProduct)).toBe("1 / (2 * 3)")
  })

  it("renders functions with comma-separated arguments", () => {
    const nodes: TraceNode[] = [
      { id: 1, kind: "function", label: "max", children: [2, 3] },
      binary(2, "+", 4, 5),
      literal(3, "9"),
      literal(4, "1"),
      literal(5, "2"),
    ]
    const steps = stepsOf(nodes)
    expect(steps.map((step) => step.expression)).toEqual(["1 + 2", "max(1 + 2, 9)"])
    expect(steps[1].kind).toBe("function")
  })

  it("behaves when no node has a value", () => {
    for (const step of stepsOf(sumsProduct.map(({ value, ...rest }) => rest))) {
      expect(step.value).toBeUndefined()
      expect("value" in step).toBe(false)
      for (const input of step.inputs) expect("value" in input).toBe(false)
    }
  })

  it("emits a shared node once", () => {
    const nodes: TraceNode[] = [
      binary(1, "*", 2, 2),
      binary(2, "+", 3, 4),
      literal(3, "1"),
      literal(4, "2"),
    ]
    const steps = stepsOf(nodes)
    expect(steps.map((step) => step.id)).toEqual([2, 1])
    expect(steps[1].expression).toBe("(1 + 2) * (1 + 2)")
  })

  it("survives a cyclic children reference", () => {
    const nodes: TraceNode[] = [
      binary(1, "+", 2, 3),
      binary(2, "*", 1, 3),
      literal(3, "5"),
    ]
    const steps = stepsOf(nodes)
    expect(steps.map((step) => step.id)).toEqual([2, 1])
    expect(steps[1].expression).toContain("…")
  })

  it("survives a self-referencing node and dangling child ids", () => {
    const selfLoop: TraceNode[] = [{ id: 1, kind: "operation", label: "+", children: [1, 99] }]
    expect(() => stepsOf(selfLoop)).not.toThrow()
    expect(stepsOf(selfLoop).map((step) => step.id)).toEqual([1])
  })

  it("caps depth and step count on a very deep chain", () => {
    const length = 2000
    const nodes: TraceNode[] = []
    for (let id = 1; id <= length; id += 1) {
      nodes.push(binary(id, "+", id + 1, length + 1))
    }
    nodes.push(literal(length + 1, "1"))
    const steps = stepsOf(nodes)
    expect(steps.length).toBeGreaterThan(0)
    expect(steps.length).toBeLessThanOrEqual(500)
    expect(Math.max(...steps.map((step) => step.depth))).toBeLessThanOrEqual(32)
  })
})
