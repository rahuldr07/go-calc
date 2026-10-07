import { describe, expect, it } from "vitest"
import { toLatex } from "../src/lib/math/latex"

describe("toLatex products", () => {
  it("renders explicit multiplication as a dot", () => {
    expect(toLatex("m_e * c^2")).toBe("m_{e}\\cdot c^{2}")
  })

  it("renders implicit multiplication between variables as a dot", () => {
    expect(toLatex("a b")).toBe("a\\cdot b")
  })

  it("keeps a number and its unit together with a thin space", () => {
    expect(toLatex("2 s")).toBe("2\\,\\mathrm{s}")
    expect(toLatex("1.5e-3 kg")).toBe("1.5\\times10^{-3}\\,\\mathrm{kg}")
  })

  it("does not read a variable after pi as a unit", () => {
    expect(toLatex("2 pi r")).toBe("2\\,\\pi\\cdot r")
  })
})

describe("toLatex fractions", () => {
  it("builds a fraction from simple operands", () => {
    expect(toLatex("1/3")).toBe("\\frac{1}{3}")
    expect(toLatex("x/y")).toBe("\\frac{x}{y}")
  })

  it("strips grouping parentheses inside a fraction", () => {
    expect(toLatex("sin(x)/(1+x^2)")).toBe("\\frac{\\sin(x)}{1+x^{2}}")
  })

  it("falls back to a slash when operands are hard to delimit", () => {
    expect(toLatex("a/b/c")).toBe("\\frac{a}{b}/c")
    expect(toLatex("a/-b")).toBe("a/-b")
  })

  it("reads a unit quotient after a number as units", () => {
    expect(toLatex("9.81 m/s^2 * 2 s")).toBe(
      "9.81\\,\\frac{\\mathrm{m}}{\\mathrm{s}^{2}}\\cdot 2\\,\\mathrm{s}",
    )
  })
})

describe("toLatex powers and subscripts", () => {
  it("braces single and multi-character exponents", () => {
    expect(toLatex("c^2")).toBe("c^{2}")
    expect(toLatex("2^10000")).toBe("2^{10000}")
    expect(toLatex("x^(n+1)")).toBe("x^{n+1}")
    expect(toLatex("s^-2")).toBe("s^{-2}")
  })

  it("nests right-associative powers", () => {
    expect(toLatex("2^3^4")).toBe("2^{3^{4}}")
  })

  it("accepts ** as a power", () => {
    expect(toLatex("x**2")).toBe("x^{2}")
  })

  it("turns underscores into subscripts", () => {
    expect(toLatex("m_e")).toBe("m_{e}")
    expect(toLatex("x_12")).toBe("x_{12}")
    expect(toLatex("T_max")).toBe("T_{\\mathrm{max}}")
  })
})

describe("toLatex functions and symbols", () => {
  it("renders roots", () => {
    expect(toLatex("sqrt(2)")).toBe("\\sqrt{2}")
    expect(toLatex("cbrt(27)")).toBe("\\sqrt[3]{27}")
  })

  it("renders standard functions with their macros", () => {
    expect(toLatex("sin(x)")).toBe("\\sin(x)")
    expect(toLatex("ln(2)")).toBe("\\ln(2)")
    expect(toLatex("exp(1)")).toBe("\\exp(1)")
    expect(toLatex("min(1, 2)")).toBe("\\min(1,\\,2)")
    expect(toLatex("max(a, b)")).toBe("\\max(a,\\,b)")
    expect(toLatex("abs(x)")).toBe("\\left|x\\right|")
  })

  it("renders plus-minus", () => {
    expect(toLatex("12.4 +- 0.2")).toBe("12.4\\pm 0.2")
  })

  it("renders a measured product", () => {
    expect(toLatex("(12.4 +- 0.2 cm) * (7.1 +- 0.1 cm)")).toBe(
      "(12.4\\pm 0.2\\,\\mathrm{cm})\\cdot (7.1\\pm 0.1\\,\\mathrm{cm})",
    )
  })

  it("does not mistake object prototype names for known symbols", () => {
    expect(toLatex("constructor")).toBe("\\mathrm{constructor}")
    expect(toLatex("toString(1)")).toBe("\\mathrm{toString}(1)")
  })

  it("renders greek constants", () => {
    expect(toLatex("pi")).toBe("\\pi")
  })

  it("sets multi-character names upright", () => {
    expect(toLatex("cm")).toBe("\\mathrm{cm}")
  })

  it("sizes parentheses to tall content", () => {
    expect(toLatex("(1/2)^2")).toBe("\\left(\\frac{1}{2}\\right)^{2}")
    expect(toLatex("(a+b)^2")).toBe("(a+b)^{2}")
  })

  it("renders factorial and percent", () => {
    expect(toLatex("5!")).toBe("5!")
    expect(toLatex("50%")).toBe("50\\%")
  })
})

describe("toLatex on malformed input", () => {
  it("returns nothing for empty input", () => {
    expect(toLatex("")).toBe("")
    expect(toLatex("   ")).toBe("")
  })

  it("falls back to escaped text for half-typed expressions", () => {
    expect(toLatex("2+")).toBe("\\text{2+}")
    expect(toLatex("(((")).toBe("\\text{(((}")
    expect(toLatex("2 * (3 + ")).toBe("\\text{2 * (3 + }")
    expect(toLatex("m_")).toBe("\\text{m\\_}")
    expect(toLatex("x^")).toBe("\\text{x\\textasciicircum{}}")
  })

  it("falls back for characters it does not know", () => {
    expect(toLatex("🙂")).toBe("\\text{🙂}")
    expect(toLatex("a $ b")).toBe("\\text{a \\$ b}")
    expect(toLatex("{\\}")).toBe("\\text{\\{\\textbackslash{}\\}}")
  })

  it("never throws on pathological input", () => {
    const deep = "(".repeat(200000)
    const garbage = ["", ")", "))((", "^^^", "1..2", "/", "+-+-", "\u0000", "a,b", "f(,)", "sqrt()", deep]
    for (const input of garbage) {
      expect(() => toLatex(input)).not.toThrow()
      expect(typeof toLatex(input)).toBe("string")
    }
  })
})
