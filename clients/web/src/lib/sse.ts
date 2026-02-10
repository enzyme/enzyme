import { getApiBase, getAuthToken, type SSEEvent, type SSEEventType } from '@feather/api-client';

type EventHandler<T extends SSEEvent = SSEEvent> = (event: T) => void;

// Extract a specific event type from the SSEEvent union by its type property
type SSEEventByType<T extends SSEEventType> = Extract<SSEEvent, { type: T }>;

export class SSEConnection {
  private eventSource: EventSource | null = null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private handlers: Map<SSEEventType | '*', EventHandler<any>[]> = new Map();
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 3000;
  private isConnecting = false;
  private workspaceId: string;
  private onDisconnect?: () => void;

  constructor(workspaceId: string) {
    this.workspaceId = workspaceId;
  }

  setOnDisconnect(callback: () => void): void {
    this.onDisconnect = callback;
  }

  connect(): void {
    if (this.eventSource || this.isConnecting) {
      return;
    }

    this.isConnecting = true;
    const token = getAuthToken();
    const url = `${getApiBase()}/workspaces/${this.workspaceId}/events${token ? `?token=${token}` : ''}`;
    this.eventSource = new EventSource(url);

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
      this.onDisconnect?.();
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

  on<T extends SSEEventType>(eventType: T, handler: EventHandler<SSEEventByType<T>>): () => void;
  on(eventType: '*', handler: EventHandler<SSEEvent>): () => void;
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
