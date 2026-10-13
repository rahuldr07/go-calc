import { describe, expect, it } from "vitest"
import { decodeSheet, encodeSheet } from "../src/lib/state/share"

describe("share links", () => {
  it("round-trip a sheet through the URL fragment", async () => {
    const sheet = "# trip\ndistance = 412 km\nfuel = 6.8 L/100km * distance in L\n√2 ± π"
    const hash = await encodeSheet(sheet)
    expect(hash.startsWith("#s=") || hash.startsWith("#p=")).toBe(true)
    expect(hash.slice(3)).not.toMatch(/[+/=]/)
    expect(await decodeSheet(hash)).toBe(sheet)
  })

  it("refuse a fragment that is not a sheet", async () => {
    expect(await decodeSheet("#s=not-a-real-sheet")).toBeNull()
    expect(await decodeSheet("#section-2")).toBeNull()
  })
})

import { scientific } from "../src/lib/editor/notepad"

describe("huge integers in the sheet", () => {
  it("show a mantissa, an exponent and the digit count", () => {
    expect(scientific("1606938044258990275541962092341162602522202993782792835301376")).toBe(
      "≈ 1.60693804426 × 10^60 · 61 digits",
    )
    expect(scientific("-1" + "0".repeat(400))).toBe("≈ -1 × 10^400 · 401 digits")
    expect(scientific("123456789012345678901234567 m")).toBe("≈ 1.23456789012 × 10^26 m · 27 digits")
  })

  it("leave ordinary numbers alone", () => {
    expect(scientific("1234567890")).toBeNull()
    expect(scientific("0.5")).toBeNull()
    expect(scientific("x = 2")).toBeNull()
  })
})

describe("rounding the mantissa", () => {
  it("carries into the exponent", () => {
    expect(scientific("9".repeat(30))).toBe("≈ 1 × 10^30 · 30 digits")
  })
})
