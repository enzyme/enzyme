// Shared helpers for K6 load tests
import http from 'k6/http';
import type {
  AuthResponse,
  MeResponse,
  ChannelWithMembership,
  MessageWithUser,
  MessageListResult,
} from '@enzyme/api-client';

// Re-export types used by test files
export type { MeResponse, MessageListResult };

// Base URL — override with K6_BASE_URL env var
export const BASE_URL = __ENV.K6_BASE_URL || 'http://localhost:8080';

// Safety guard: refuse to run against non-localhost targets without explicit opt-in
const isLocalhost =
  BASE_URL.startsWith('http://localhost') || BASE_URL.startsWith('http://127.0.0.1');
if (!isLocalhost && !__ENV.K6_CONFIRM_REMOTE) {
  throw new Error(
    `Refusing to run load tests against non-localhost target: ${BASE_URL}. ` +
      `Set --env K6_CONFIRM_REMOTE=1 to override.`,
  );
}

const API = `${BASE_URL}/api`;

// Response shapes not directly in the OpenAPI schema (endpoint-specific wrappers)
export interface ChannelListResponse {
  channels: ChannelWithMembership[];
}

export interface SendMessageResponse {
  message: MessageWithUser;
}

// Shared constants
export const REACTION_EMOJIS = ['+1', 'heart', 'rocket', 'eyes', 'fire', 'tada', '100', 'wave'];

export const SEARCH_QUERIES = ['hello', 'test', 'meeting', 'update', 'help', 'thanks'];

export function pickRandom<T>(arr: readonly T[]): T {
  if (arr.length === 0) throw new Error('pickRandom called with empty array');
  return arr[Math.floor(Math.random() * arr.length)];
}

// K6's res.json() returns a broad union type. This helper provides a clean
// typed cast without needing `as unknown as T` at every call site.
export function jsonAs<T>(val: unknown): T {
  return val as T;
}

// Seed users (all have password "password")
export const SEED_USERS = [
  { email: 'alice@example.com', name: 'Alice Chen' },
  { email: 'bob@example.com', name: 'Bob Martinez' },
  { email: 'carol@example.com', name: 'Carol Williams' },
  { email: 'dave@example.com', name: 'Dave Johnson' },
  { email: 'eve@example.com', name: 'Eve Kim' },
  { email: 'frank@example.com', name: "Frank O'Brien" },
  { email: 'grace@example.com', name: 'Grace Patel' },
  { email: 'hank@example.com', name: 'Hank Nguyen' },
] as const;

const PASSWORD = 'password';

export interface UserContext {
  email: string;
  name: string;
  token: string;
  workspaceId: string;
  channels: string[];
}

// JSON request helper
export function jsonHeaders(token?: string) {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return { headers };
}

// Login and return token
export function login(email: string): string | null {
  const res = http.post(
    `${API}/auth/login`,
    JSON.stringify({ email, password: PASSWORD }),
    jsonHeaders(),
  );
  if (res.status !== 200) {
    console.error(`Login failed for ${email}: ${res.status} ${res.body}`);
    return null;
  }
  return jsonAs<AuthResponse>(res.json()).token;
}

// Register a unique user and return token
interface RegisterResult {
  token: string | null;
  email: string;
}

export function registerUser(suffix: string): RegisterResult {
  const email = `loadtest-${suffix}-${Date.now()}@example.com`;
  const res = http.post(
    `${API}/auth/register`,
    JSON.stringify({
      email,
      password: PASSWORD,
      display_name: `LoadTest ${suffix}`,
    }),
    jsonHeaders(),
  );
  if (res.status !== 200) {
    console.error(`Register failed: ${res.status} ${res.body}`);
    return { token: null, email };
  }
  return { token: jsonAs<AuthResponse>(res.json()).token, email };
}

// Login all seed users once and resolve their workspace/channel context.
// Call this from setup() so tokens are reused across VUs without hitting rate limits.
export function loginAllUsers(): UserContext[] {
  const users: UserContext[] = [];
  for (const user of SEED_USERS) {
    const token = login(user.email);
    if (!token) continue;

    const meRes = getMe(token);
    if (meRes.status !== 200) continue;

    const me = jsonAs<MeResponse>(meRes.json());
    if (!me.workspaces?.length) continue;

    const workspaceId = me.workspaces[0].id;

    const chRes = listChannels(token, workspaceId);
    let channels: string[] = [];
    if (chRes.status === 200) {
      const data = jsonAs<ChannelListResponse>(chRes.json());
      channels = data.channels.filter((c) => c.type === 'public').map((c) => c.id);
    }

    users.push({
      email: user.email,
      name: user.name,
      token,
      workspaceId,
      channels,
    });
  }
  return users;
}

// Pick a user context from the setup data based on VU number
export function pickUser(setupData: UserContext[]): UserContext {
  if (setupData.length === 0) {
    throw new Error('No users available -- is the seed data loaded?');
  }
  return setupData[__VU % setupData.length];
}

// Get current user
export function getMe(token: string) {
  return http.get(`${API}/auth/me`, jsonHeaders(token));
}

// List channels in a workspace
export function listChannels(token: string, workspaceId: string) {
  return http.post(`${API}/workspaces/${workspaceId}/channels/list`, null, jsonHeaders(token));
}

// Send a message
export function sendMessage(token: string, channelId: string, content: string) {
  return http.post(
    `${API}/channels/${channelId}/messages/send`,
    JSON.stringify({ content }),
    jsonHeaders(token),
  );
}

// List messages
export function listMessages(token: string, channelId: string, limit = 50) {
  return http.post(
    `${API}/channels/${channelId}/messages/list`,
    JSON.stringify({ limit }),
    jsonHeaders(token),
  );
}

// Add reaction
export function addReaction(token: string, messageId: string, emoji: string) {
  return http.post(
    `${API}/messages/${messageId}/reactions/add`,
    JSON.stringify({ emoji }),
    jsonHeaders(token),
  );
}

// Typing indicator
export function startTyping(token: string, workspaceId: string, channelId: string) {
  return http.post(
    `${API}/workspaces/${workspaceId}/typing/start`,
    JSON.stringify({ channel_id: channelId }),
    jsonHeaders(token),
  );
}

// Search messages
export function searchMessages(token: string, workspaceId: string, query: string) {
  return http.post(
    `${API}/workspaces/${workspaceId}/messages/search`,
    JSON.stringify({ query }),
    jsonHeaders(token),
  );
}

// Standard thresholds used across tests
export const STANDARD_THRESHOLDS = {
  http_req_failed: ['rate<0.01'],
  http_req_duration: ['p(95)<500', 'p(99)<1000'],
};
