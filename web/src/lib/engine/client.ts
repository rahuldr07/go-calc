import { SUPERSEDED, type Derivation, type Line, type Request, type Sheet } from "./schema"

interface Reply {
  id: number
  ok: boolean
  answer?: string
  error?: string
}

interface Waiting {
  resolve(sheet: Sheet): void
  reject(err: Error): void
}

export type EngineStatus = "loading" | "ready" | "failed"

function siteRoot(): string {
  if (typeof location === "undefined") return "/"
  return new URL(import.meta.env?.BASE_URL ?? "/", location.href).href
}

export class Engine {
  status: EngineStatus = "loading"
  readonly ready: Promise<void>
  private readonly worker: Worker
  private readonly waiting = new Map<number, Waiting>()
  private readonly rawWaiting = new Map<number, (answer: string) => void>()
  private issued = 0
  private newest = 0

  constructor(options: { url?: string; base?: string; worker?: Worker } = {}) {
    this.worker = options.worker ?? new Worker(new URL("./worker.ts", import.meta.url))
    this.worker.onmessage = (event: MessageEvent<Reply>) => this.settle(event.data)
    this.worker.onerror = (event) => this.fail(event.message || "the engine worker stopped")
    const base = options.base ?? siteRoot()
    this.ready = this.dispatch({ kind: "start", base, url: options.url }).then(
      () => {
        this.status = "ready"
      },
      (err: Error) => {
        this.status = "failed"
        console.error("go-calc engine failed to start:", err)
        throw err
      },
    )
  }

  evaluate(request: Request): Promise<Sheet> {
    return this.dispatch({ kind: "evaluate", request: JSON.stringify(request) }, true)
  }

  vocabulary(): Promise<string> {
    return this.raw({ kind: "vocabulary" })
  }

  async steps(input: string, line: Line): Promise<Derivation | null> {
    const answer = await this.raw({ kind: "steps", input, request: JSON.stringify(line) })
    if (answer === "") return null
    try {
      return JSON.parse(answer) as Derivation
    } catch {
      return null
    }
  }

  private raw(message: { kind: "vocabulary" | "steps"; input?: string; request?: string }): Promise<string> {
    const id = ++this.issued
    return new Promise<string>((resolve) => {
      this.rawWaiting.set(id, resolve)
      this.worker.postMessage({ id, ...message })
    })
  }

  terminate() {
    this.worker.terminate()
  }

  private dispatch(
    message: { kind: "start" | "evaluate"; base?: string; url?: string; request?: string },
    ordered = false,
  ): Promise<Sheet> {
    const id = ++this.issued
    if (ordered) this.newest = id
    return new Promise<Sheet>((resolve, reject) => {
      this.waiting.set(id, { resolve, reject })
      this.worker.postMessage({ id, ...message })
    })
  }

  private settle(reply: Reply) {
    const raw = this.rawWaiting.get(reply.id)
    if (raw) {
      this.rawWaiting.delete(reply.id)
      raw(reply.ok ? (reply.answer ?? "") : "")
      return
    }
    const pending = this.waiting.get(reply.id)
    if (!pending) return
    this.waiting.delete(reply.id)
    if (!reply.ok) return pending.reject(new Error(reply.error ?? "the engine failed"))
    if (reply.answer === undefined) return pending.resolve({ schema: "", lines: [] })
    if (reply.id < this.newest) return pending.reject(new Error(SUPERSEDED))
    try {
      pending.resolve(JSON.parse(reply.answer) as Sheet)
    } catch {
      pending.reject(new Error("the engine sent something that is not a sheet"))
    }
  }

  private fail(message: string) {
    this.status = "failed"
    for (const [id, raw] of this.rawWaiting) {
      this.rawWaiting.delete(id)
      raw("")
    }
    for (const [id, pending] of this.waiting) {
      this.waiting.delete(id)
      pending.reject(new Error(message))
    }
  }
}
