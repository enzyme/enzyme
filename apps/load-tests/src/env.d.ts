// Type declarations for K6 community extensions

declare module 'k6/x/sse' {
  interface SSEEvent {
    id: string;
    name: string;
    data: string;
  }

  interface SSEClient {
    on(event: 'open', callback: () => void): void;
    on(event: 'event', callback: (event: SSEEvent) => void): void;
    on(event: 'error', callback: (error: Error) => void): void;
  }

  interface SSEParams {
    headers?: Record<string, string>;
    tags?: Record<string, string>;
    timeout?: string;
  }

  interface SSEResponse {
    status: number;
    error: string;
  }

  function open(url: string, params: SSEParams, callback: (client: SSEClient) => void): SSEResponse;

  function open(url: string, callback: (client: SSEClient) => void): SSEResponse;

  export default { open };
}
