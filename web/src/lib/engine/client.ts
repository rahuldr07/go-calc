import { SUPERSEDED, type Request, type Sheet } from "./schema"

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

export class Engine {
  status: EngineStatus = "loading"
  readonly ready: Promise<void>
  private readonly worker: Worker
  private readonly waiting = new Map<number, Waiting>()
  private issued = 0
  private newest = 0

  constructor(options: { url?: string; worker?: Worker } = {}) {
    this.worker =
      options.worker ?? new Worker(new URL("./worker.ts", import.meta.url), { type: "module" })
    this.worker.onmessage = (event: MessageEvent<Reply>) => this.settle(event.data)
    this.worker.onerror = (event) => this.fail(event.message || "the engine worker stopped")
    this.ready = this.dispatch({ kind: "start", url: options.url }).then(
      () => {
        this.status = "ready"
      },
      (err: Error) => {
        this.status = "failed"
        throw err
      },
    )
  }

  evaluate(request: Request): Promise<Sheet> {
    return this.dispatch({ kind: "evaluate", request: JSON.stringify(request) }, true)
  }

  terminate() {
    this.worker.terminate()
  }

  private dispatch(
    message: { kind: "start" | "evaluate"; url?: string; request?: string },
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
    for (const [id, pending] of this.waiting) {
      this.waiting.delete(id)
      pending.reject(new Error(message))
    }
  }
}
