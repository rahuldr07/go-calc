declare const Go: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): void }
declare function importScripts(...urls: string[]): void
declare function goCalcEval(request: string): string

interface Incoming {
  id: number
  kind: "start" | "evaluate" | "vocabulary" | "steps"
  base?: string
  url?: string
  request?: string
  input?: string
}

let running: Promise<void> | null = null

async function start(base: string, url?: string) {
  importScripts(new URL("wasm_exec.js", base).href)
  const go = new Go()
  const answer = await fetch(url ?? new URL("gocalc.wasm", base).href)
  if (!answer.ok) throw new Error(`the engine could not be fetched (${answer.status})`)
  const bytes = await answer.arrayBuffer()
  const magic = new Uint8Array(bytes, 0, Math.min(4, bytes.byteLength))
  if (magic[0] !== 0x00 || magic[1] !== 0x61 || magic[2] !== 0x73 || magic[3] !== 0x6d) {
    throw new Error("the engine was served as something other than WebAssembly")
  }
  const wasm = await WebAssembly.instantiate(bytes, go.importObject)
  void go.run(wasm.instance)
}

self.onmessage = async (event: MessageEvent<Incoming>) => {
  const { id, kind, base, url, request, input } = event.data
  try {
    if (kind === "start") {
      running = start(base ?? "/", url)
      await running
      self.postMessage({ id, ok: true })
      return
    }
    if (!running) throw new Error("the engine was never started")
    await running
    if (kind === "vocabulary") {
      const read = (self as unknown as { goCalcVocabulary?: () => string }).goCalcVocabulary
      self.postMessage({ id, ok: true, answer: read ? read() : "" })
      return
    }
    if (kind === "steps") {
      const work = (self as unknown as { goCalcSteps?: (input: string, line: string) => string }).goCalcSteps
      self.postMessage({ id, ok: true, answer: work ? work(input ?? "", request ?? "") : "" })
      return
    }
    self.postMessage({ id, ok: true, answer: goCalcEval(request ?? "") })
  } catch (err) {
    self.postMessage({ id, ok: false, error: err instanceof Error ? err.message : String(err) })
  }
}
