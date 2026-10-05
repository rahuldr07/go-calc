import type { Line } from "../engine/schema"

export type VizKind =
  | "measured-rectangle"
  | "square"
  | "circle"
  | "sampled-signal"
  | "derivative"
  | "integral"
  | "measurement-interval"

export type VizSource = "engine-fields" | "engine-points" | "engine-interval" | "contract"

export interface Quantity {
  label: string
  text: string
  sigma?: string
}

export interface SamplePacket {
  points: { x: number; y: number }[]
  lower?: { x: number; y: number }[]
  upper?: { x: number; y: number }[]
  xLabel?: string
  yLabel?: string
  at?: number
  bounds?: { a: number; b: number }
}

export interface VizPacket {
  kind: VizKind
  source: VizSource
  quantity: string
  dimensions: string
  inputs: Quantity[]
  samples?: SamplePacket
  story: { observe: string; operation: string; meaning: string }
}

export interface Candidate {
  line: Line
  expression: string
}

export type Detector = (c: Candidate) => VizPacket | null
