import { describe, expect, it } from "vitest"
import {
  digitCount,
  dimensionFormula,
  groupDigits,
  headlineFor,
  logDomain,
  magnitudeOf,
  niceTicks,
  normalizeShares,
  orderedDimensions,
  placeOn,
  ratioOf,
  ratioSentence,
  scientificLabel,
  shareLabel,
} from "../src/lib/viz/figures"

const power = (base: bigint, exponent: number): string => (base ** BigInt(exponent)).toString()

describe("dimensions", () => {
  it("orders canonically and drops zero exponents", () => {
    const terms = orderedDimensions({ T: -2, M: 1, L: 2, I: 0 })
    expect(terms.map((t) => t.symbol)).toEqual(["L", "M", "T"])
    expect(dimensionFormula(terms)).toBe("L² M / T²")
  })
  it("writes a pure denominator and dimensionless", () => {
    expect(dimensionFormula(orderedDimensions({ T: -1 }))).toBe("1 / T")
    expect(dimensionFormula(orderedDimensions({}))).toBe("dimensionless")
    expect(dimensionFormula(orderedDimensions(undefined))).toBe("dimensionless")
  })
})

describe("big integers", () => {
  it("counts digits without Number", () => {
    expect(digitCount(power(2n, 10000))).toBe(3011)
    expect(digitCount("-0042")).toBe(2)
    expect(groupDigits("1234567")).toBe("1,234,567")
    expect(groupDigits("3011")).toBe("3,011")
  })
  it("reads 2^10000 as a digit count", () => {
    const magnitude = magnitudeOf({ exact: power(2n, 10000) })
    expect(magnitude?.wholeDigits).toBe(3011)
    expect(magnitude?.log10).toBeCloseTo(10000 * Math.log10(2), 9)
    expect(headlineFor(magnitude!)).toBe("3,011 digits")
  })
  it("reads small decimals and rationals", () => {
    const small = magnitudeOf({ decimal: "8.19e-14" })
    expect(small?.log10).toBeCloseTo(Math.log10(8.19e-14), 12)
    expect(scientificLabel(small!.log10)).toBe("8.19 × 10⁻¹⁴")
    const third = magnitudeOf({ exact: "1/3" })
    expect(third?.log10).toBeCloseTo(Math.log10(1 / 3), 12)
    expect(third?.isInteger).toBe(false)
    expect(magnitudeOf({ exact: "0" })?.sign).toBe(0)
    expect(magnitudeOf({ exact: "-5" })?.sign).toBe(-1)
    expect(magnitudeOf({ decimal: "nonsense" })).toBeNull()
  })
  it("rolls a rounded mantissa into the next decade", () => {
    expect(scientificLabel(Math.log10(9.9999))).toBe("1.00 × 10¹")
  })
})

describe("log axis", () => {
  it("covers value and 1, with a nice tick on 0", () => {
    const [lo, hi] = logDomain(-13.087, 4)
    expect(lo).toBeLessThanOrEqual(-14)
    expect(hi).toBeGreaterThanOrEqual(0)
    const ticks = niceTicks(lo, hi, 5)
    expect(ticks.some((t) => t.isReference && t.value === 0)).toBe(true)
    expect(ticks.length).toBeLessThanOrEqual(8)
  })
  it("keeps a minimum span for values near 1", () => {
    const [lo, hi] = logDomain(-0.477, 4)
    expect(hi - lo).toBeGreaterThanOrEqual(4)
  })
  it("places on the axis and clamps", () => {
    expect(placeOn(0, -10, 10, 16, 324)).toBe(170)
    expect(placeOn(99, -10, 10, 16, 324)).toBe(324)
  })
})

describe("ratio", () => {
  it("measures a small fraction", () => {
    const reading = ratioOf("2201", "250000")!
    expect(reading.fraction).toBeCloseTo(0.008804, 6)
    expect(ratioSentence(reading)).toBe("the numerator is ≈ 0.88% of the denominator")
  })
  it("survives thousand-digit integers", () => {
    const reading = ratioOf(power(3n, 8000), power(3n, 8001))!
    expect(reading.fraction).toBeCloseTo(1 / 3, 9)
  })
  it("handles improper, zero and invalid", () => {
    expect(ratioOf("7", "3")?.numeratorIsLarger).toBe(true)
    expect(ratioOf("0", "3")?.numeratorIsZero).toBe(true)
    expect(ratioOf("1", "0")).toBeNull()
  })
})

describe("budget", () => {
  it("normalises, ranks and labels shares", () => {
    const rows = normalizeShares([
      { name: "a", share: 0.002, sigma: 1, sensitivity: 1 },
      { name: "b", share: 0.99, sigma: 1, sensitivity: 1 },
      { name: "c", share: 0.008, sigma: 1, sensitivity: 1 },
    ])
    expect(rows.map((r) => r.name)).toEqual(["b", "c", "a"])
    expect(rows.reduce((s, r) => s + r.share, 0)).toBeCloseTo(1, 12)
    expect(shareLabel(0.00004)).toBe("<0.1%")
    expect(shareLabel(0.9996)).toBe(">99.9%")
    expect(shareLabel(0.5)).toBe("50.0%")
  })
})
