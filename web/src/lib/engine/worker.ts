declare const Go: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): void }
declare function goCalcEval(request: string): string

interface Incoming {
  id: number
  kind: "start" | "evaluate"
  url?: string
  request?: string
}

let running: Promise<void> | null = null

async function start(url: string) {
  const loader = new URL("wasm_exec.js", self.location.href).href
  ;(self as unknown as { importScripts(...urls: string[]): void }).importScripts(loader)
  const go = new Go()
  const answer = await fetch(url)
  if (!answer.ok) throw new Error(`the engine could not be fetched (${answer.status})`)
  const wasm = await WebAssembly.instantiate(await answer.arrayBuffer(), go.importObject)
  void go.run(wasm.instance)
}

self.onmessage = async (event: MessageEvent<Incoming>) => {
  const { id, kind, url, request } = event.data
  try {
    if (kind === "start") {
      running = start(url ?? "gocalc.wasm")
      await running
      self.postMessage({ id, ok: true })
      return
    }
    if (!running) throw new Error("the engine was never started")
    await running
    self.postMessage({ id, ok: true, answer: goCalcEval(request ?? "") })
  } catch (err) {
    self.postMessage({ id, ok: false, error: err instanceof Error ? err.message : String(err) })
  }
}
