import { useCallback, useEffect, useRef, useState } from "react"

import {
  getProvisioningAuthHeaders,
  hasProvisioningToken,
  notifyProvisioningAuthRequired,
} from "@/api/provisioning"
import type { DeviceStatus } from "@/api/provisioning"

interface ProvisioningEvent {
  type: string
  level: string
  message: string
  timestamp: number
  details?: Record<string, unknown>
  status?: DeviceStatus
}

interface UseProvisioningSSEOptions {
  onEvent?: (event: ProvisioningEvent) => void
  onStatusChange?: (status: DeviceStatus) => void
  onError?: (error: Error) => void
}

function parseSSEBlock(block: string): { event: string; data: string } | null {
  const lines = block.split("\n")
  let eventName = ""
  const dataLines: string[] = []
  for (const line of lines) {
    if (line.startsWith(":")) {
      continue
    }
    if (line.startsWith("event:")) {
      eventName = line.slice(6).trim()
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).replace(/^\s/, ""))
    }
  }
  const data = dataLines.join("\n")
  if (!data) {
    return null
  }
  return { event: eventName || "message", data }
}

async function readProvisioningSSE(
  signal: AbortSignal,
  onParsed: (eventName: string, data: string) => void
): Promise<void> {
  const res = await fetch("/api/provisioning/events", {
    method: "GET",
    headers: {
      Accept: "text/event-stream",
      ...getProvisioningAuthHeaders(),
    },
    signal,
  })
  if (!res.ok) {
    if (res.status === 401) {
      notifyProvisioningAuthRequired()
    }
    throw new Error(`SSE HTTP ${res.status}`)
  }
  if (!res.body) {
    throw new Error("SSE missing body")
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ""
  for (;;) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    buf += decoder.decode(value, { stream: true })
    for (;;) {
      const sep = buf.indexOf("\n\n")
      if (sep < 0) {
        break
      }
      const block = buf.slice(0, sep)
      buf = buf.slice(sep + 2)
      const parsed = parseSSEBlock(block)
      if (parsed) {
        onParsed(parsed.event, parsed.data)
      }
    }
  }
}

export function useProvisioningSSE(options: UseProvisioningSSEOptions = {}) {
  const optsRef = useRef(options)

  const [isConnected, setIsConnected] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)
  const fetchAbortRef = useRef<AbortController | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const connectRef = useRef<() => (() => void) | void>(() => {})

  const connect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }

    fetchAbortRef.current?.abort()
    fetchAbortRef.current = null

    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }

    if (hasProvisioningToken()) {
      const ac = new AbortController()
      fetchAbortRef.current = ac
      ;(async () => {
        try {
          await readProvisioningSSE(ac.signal, (eventName, data) => {
            const { onEvent, onStatusChange } = optsRef.current
            if (eventName === "connected") {
              setIsConnected(true)
              return
            }
            if (eventName !== "device") {
              return
            }
            try {
              const ev = JSON.parse(data) as ProvisioningEvent
              onEvent?.(ev)
              if (ev.status) {
                onStatusChange?.(ev.status)
              }
            } catch {
              // ignore parse errors
            }
          })
        } catch (e) {
          if (
            e instanceof Error &&
            (e.name === "AbortError" || e.message === "The user aborted a request.")
          ) {
            return
          }
          setIsConnected(false)
          optsRef.current.onError?.(
            e instanceof Error ? e : new Error("SSE connection failed")
          )
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connectRef.current()
          }, 3000)
          return
        }
        if (!ac.signal.aborted) {
          setIsConnected(false)
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connectRef.current()
          }, 3000)
        }
      })()
      return () => {
        ac.abort()
        setIsConnected(false)
      }
    }

    const eventSource = new EventSource("/api/provisioning/events")
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setIsConnected(true)
    }

    eventSource.onerror = () => {
      setIsConnected(false)
      eventSource.close()

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      reconnectTimeoutRef.current = window.setTimeout(() => {
        connectRef.current()
      }, 3000)

      optsRef.current.onError?.(new Error("SSE connection failed"))
    }

    eventSource.addEventListener("connected", () => {
      setIsConnected(true)
    })

    eventSource.addEventListener("device", (event) => {
      try {
        const data = JSON.parse(event.data) as ProvisioningEvent
        optsRef.current.onEvent?.(data)
        if (data.status) {
          optsRef.current.onStatusChange?.(data.status)
        }
      } catch {
        // Ignore parse errors
      }
    })

    return () => {
      eventSource.close()
      setIsConnected(false)
    }
  }, [])

  useEffect(() => {
    optsRef.current = options
    connectRef.current = connect
  }, [options, connect])

  useEffect(() => {
    const cleanup = connect()
    return () => {
      cleanup()
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      fetchAbortRef.current?.abort()
      fetchAbortRef.current = null
    }
  }, [connect])

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    fetchAbortRef.current?.abort()
    fetchAbortRef.current = null
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    setIsConnected(false)
  }, [])

  return { isConnected, connect, disconnect }
}
