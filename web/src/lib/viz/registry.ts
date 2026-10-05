import type { Candidate, Detector, VizPacket } from "./types"

function inputsOf(c: Candidate) {
  return (c.line.trace ?? [])
    .filter((n) => n.kind === "input")
    .map((n) => ({ label: n.label, text: n.detail ?? "", sigma: sigmaOf(c, n.label) }))
}

function sigmaOf(c: Candidate, name: string) {
  const item = c.line.uncertainty?.budget?.find((b) => b.name === name)
  return item ? String(item.sigma) : undefined
}

function unitsOf(line: Candidate["line"]) {
  const parts = line.text.split(" ")
  return parts.length > 1 ? parts[parts.length - 1] : ""
}

const product: Detector = (c) => {
  const root = c.line.trace?.[0]
  if (!root || root.kind !== "operation" || (root.label !== "*" && root.label !== "·")) return null
  const inputs = inputsOf(c)
  if (inputs.length !== 2) return null
  const same = inputs[0].label === inputs[1].label
  return {
    kind: same ? "square" : "measured-rectangle",
    source: "engine-fields",
    quantity: c.line.text,
    dimensions: `${sideUnit(inputs[0].text)} × ${sideUnit(inputs[1].text)} → ${unitsOf(c.line)}`,
    inputs,
    story: {
      observe: `two measured lengths, ${inputs[0].label} and ${inputs[1].label}`,
      operation: "multiplied to give an area",
      meaning: dominant(c) ?? "the area the two measurements enclose",
    },
  }
}

function sideUnit(detail: string) {
  const parts = detail.split(" ")
  return parts[parts.length - 1] || "?"
}

function dominant(c: Candidate) {
  const budget = c.line.uncertainty?.budget
  if (!budget || budget.length < 2) return null
  const top = [...budget].sort((a, b) => b.share - a.share)[0]
  return `${top.name} carries most of the uncertainty, ${(top.share * 100).toFixed(1)} %`
}

const interval: Detector = (c) => {
  if (!c.line.uncertainty) return null
  return {
    kind: "measurement-interval",
    source: "engine-interval",
    quantity: c.line.text,
    dimensions: unitsOf(c.line) || "dimensionless",
    inputs: inputsOf(c),
    story: {
      observe: "a measured quantity",
      operation: "reported with its standard uncertainty",
      meaning: "the engine states how precisely this is known — no distribution is assumed",
    },
  }
}

const DETECTORS: Detector[] = [product, interval]

export function packetFor(c: Candidate): VizPacket | null {
  for (const detect of DETECTORS) {
    const packet = detect(c)
    if (packet) return packet
  }
  return null
}
