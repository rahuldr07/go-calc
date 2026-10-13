const MARK = "#s="
const PLAIN = "#p="

function toBase64Url(bytes: Uint8Array): string {
  let text = ""
  for (const b of bytes) text += String.fromCharCode(b)
  return btoa(text).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")
}

function fromBase64Url(text: string): Uint8Array {
  const padded = text.replace(/-/g, "+").replace(/_/g, "/") + "=".repeat((4 - (text.length % 4)) % 4)
  const raw = atob(padded)
  const out = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i)
  return out
}

async function through(bytes: Uint8Array, stream: GenerateTransformStream): Promise<Uint8Array> {
  const piped = new Blob([bytes as BlobPart]).stream().pipeThrough(stream)
  return new Uint8Array(await new Response(piped).arrayBuffer())
}

type GenerateTransformStream = TransformStream<Uint8Array, Uint8Array>

function compressible(): boolean {
  return typeof CompressionStream !== "undefined" && typeof DecompressionStream !== "undefined"
}

export async function encodeSheet(source: string): Promise<string> {
  const bytes = new TextEncoder().encode(source)
  if (!compressible()) return PLAIN + toBase64Url(bytes)
  const packed = await through(bytes, new CompressionStream("deflate-raw") as GenerateTransformStream)
  return MARK + toBase64Url(packed)
}

export async function decodeSheet(hash: string): Promise<string | null> {
  try {
    if (hash.startsWith(PLAIN)) return new TextDecoder().decode(fromBase64Url(hash.slice(PLAIN.length)))
    if (!hash.startsWith(MARK) || !compressible()) return null
    const bytes = fromBase64Url(hash.slice(MARK.length))
    const unpacked = await through(bytes, new DecompressionStream("deflate-raw") as GenerateTransformStream)
    return new TextDecoder("utf-8", { fatal: true }).decode(unpacked)
  } catch {
    return null
  }
}

export function sharedHash(): string {
  if (typeof location === "undefined") return ""
  return location.hash.startsWith(MARK) || location.hash.startsWith(PLAIN) ? location.hash : ""
}

export function linkFor(hash: string): string {
  return location.href.replace(/#.*$/, "") + hash
}
