import type { SSEEvent, SSEEventType } from '@feather/api-client';

type EventHandler = (event: SSEEvent) => void;

export class SSEConnection {
  private eventSource: EventSource | null = null;
  private handlers: Map<SSEEventType | '*', EventHandler[]> = new Map();
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 3000;
  private isConnecting = false;
  private workspaceId: string;

  constructor(workspaceId: string) {
    this.workspaceId = workspaceId;
  }

  connect(): void {
    if (this.eventSource || this.isConnecting) {
      return;
    }

    this.isConnecting = true;
    const url = `/api/workspaces/${this.workspaceId}/events`;
    this.eventSource = new EventSource(url, { withCredentials: true });

    this.eventSource.onopen = () => {
      this.isConnecting = false;
      console.log('[SSE] Connected to', url);
    };

    this.eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as SSEEvent;
        this.dispatch(data);
      } catch (e) {
        console.error('[SSE] Failed to parse event:', e);
      }
    };

    this.eventSource.onerror = () => {
      console.error('[SSE] Connection error, reconnecting...');
      this.disconnect();
      this.scheduleReconnect();
    };
  }

  disconnect(): void {
    this.isConnecting = false;
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimeout) {
      return;
    }
    this.reconnectTimeout = setTimeout(() => {
      this.reconnectTimeout = null;
      this.connect();
    }, this.reconnectDelay);
  }

  on(eventType: SSEEventType | '*', handler: EventHandler): () => void {
    const handlers = this.handlers.get(eventType) || [];
    handlers.push(handler);
    this.handlers.set(eventType, handlers);

    return () => {
      const idx = handlers.indexOf(handler);
      if (idx !== -1) {
        handlers.splice(idx, 1);
      }
    };
  }

  private dispatch(event: SSEEvent): void {
    // Call specific handlers
    const specificHandlers = this.handlers.get(event.type) || [];
    specificHandlers.forEach((handler) => handler(event));

    // Call wildcard handlers
    const wildcardHandlers = this.handlers.get('*') || [];
    wildcardHandlers.forEach((handler) => handler(event));
  }

  get isConnected(): boolean {
    return this.eventSource?.readyState === EventSource.OPEN;
  }
}
