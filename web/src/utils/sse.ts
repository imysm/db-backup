/**
 * SSE (Server-Sent Events) 客户端工具
 */

export interface SSEClienOptions {
  onConnected?: () => void
  onLog?: (log: string) => void
  onDisconnected?: (reason?: string) => void
  onError?: (error: Event) => void
  onPing?: () => void
}

export class SSEClient {
  private eventSource: EventSource | null = null
  private url: string
  private options: SSEClienOptions
  private reconnectAttempts = 0
  private maxReconnectAttempts = 3
  private reconnectDelay = 3000
  private manualClose = false

  constructor(url: string, options: SSEClienOptions = {}) {
    this.url = url
    this.options = options
  }

  connect() {
    if (this.eventSource) {
      this.close()
    }

    this.manualClose = false
    this.eventSource = new EventSource(this.url)

    this.eventSource.onopen = () => {
      this.reconnectAttempts = 0
      this.options.onConnected?.()
    }

    this.eventSource.onmessage = (event) => {
      // 解析事件类型
      const data = event.data
      this.options.onLog?.(data)
    }

    this.eventSource.addEventListener('connected', (event) => {
      this.options.onConnected?.()
    })

    this.eventSource.addEventListener('log', (event) => {
      this.options.onLog?.(event.data)
    })

    this.eventSource.addEventListener('ping', () => {
      this.options.onPing?.()
    })

    this.eventSource.addEventListener('disconnected', (event) => {
      const reason = event.data
      this.options.onDisconnected?.(reason)
      if (!this.manualClose && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++
        setTimeout(() => this.connect(), this.reconnectDelay)
      }
    })

    this.eventSource.onerror = (error) => {
      this.options.onError?.(error)
      if (!this.manualClose && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++
        setTimeout(() => this.connect(), this.reconnectDelay)
      }
    }
  }

  close() {
    this.manualClose = true
    if (this.eventSource) {
      this.eventSource.close()
      this.eventSource = null
    }
  }

  isConnected(): boolean {
    return this.eventSource?.readyState === EventSource.OPEN
  }
}

export default SSEClient
