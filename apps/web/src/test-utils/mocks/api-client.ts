export function mockResponse<T>(data: T) {
  return { data, response: new Response() };
}
