// Shared helpers for K6 load tests
import http from "k6/http";

// Base URL — override with K6_BASE_URL env var
// Defaults to local dev server; set to https://chat.enzyme.im for production testing
export const BASE_URL = __ENV.K6_BASE_URL || "http://localhost:8080";
const API = `${BASE_URL}/api`;

// Seed users (all have password "password")
export const SEED_USERS = [
  { email: "alice@example.com", name: "Alice Chen" },
  { email: "bob@example.com", name: "Bob Martinez" },
  { email: "carol@example.com", name: "Carol Williams" },
  { email: "dave@example.com", name: "Dave Johnson" },
  { email: "eve@example.com", name: "Eve Kim" },
  { email: "frank@example.com", name: "Frank O'Brien" },
  { email: "grace@example.com", name: "Grace Patel" },
  { email: "hank@example.com", name: "Hank Nguyen" },
];

const PASSWORD = "password";

// JSON request helper
export function jsonHeaders(token) {
  const headers = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;
  return { headers };
}

// Login and return token
export function login(email) {
  const res = http.post(
    `${API}/auth/login`,
    JSON.stringify({ email, password: PASSWORD }),
    jsonHeaders()
  );
  if (res.status !== 200) {
    console.error(`Login failed for ${email}: ${res.status} ${res.body}`);
    return null;
  }
  return res.json().token;
}

// Register a unique user and return token
export function registerUser(suffix) {
  const email = `loadtest-${suffix}-${Date.now()}@example.com`;
  const res = http.post(
    `${API}/auth/register`,
    JSON.stringify({
      email,
      password: PASSWORD,
      display_name: `LoadTest ${suffix}`,
    }),
    jsonHeaders()
  );
  if (res.status !== 200) {
    console.error(`Register failed: ${res.status} ${res.body}`);
    return { token: null, email };
  }
  return { token: res.json().token, email };
}

// Login all seed users once and resolve their workspace/channel context.
// Call this from setup() so tokens are reused across VUs without hitting rate limits.
export function loginAllUsers() {
  const users = [];
  for (const user of SEED_USERS) {
    const token = login(user.email);
    if (!token) continue;

    // GET /auth/me returns user + workspaces
    const meRes = http.get(`${API}/auth/me`, jsonHeaders(token));
    if (meRes.status !== 200) continue;

    const me = meRes.json();
    const workspaces = me.workspaces || [];
    if (workspaces.length === 0) continue;

    const workspaceId = workspaces[0].id;

    // POST /workspaces/{wid}/channels/list (no body needed)
    const chRes = http.post(
      `${API}/workspaces/${workspaceId}/channels/list`,
      null,
      jsonHeaders(token)
    );
    let channels = [];
    if (chRes.status === 200) {
      channels = (chRes.json().channels || [])
        .filter((c) => c.type === "public")
        .map((c) => c.id);
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
export function pickUser(setupData) {
  return setupData[__VU % setupData.length];
}

// Get current user
export function getMe(token) {
  return http.get(`${API}/auth/me`, jsonHeaders(token));
}

// List channels in a workspace
export function listChannels(token, workspaceId) {
  return http.post(
    `${API}/workspaces/${workspaceId}/channels/list`,
    null,
    jsonHeaders(token)
  );
}

// Send a message — POST /channels/{id}/messages/send
export function sendMessage(token, channelId, content) {
  return http.post(
    `${API}/channels/${channelId}/messages/send`,
    JSON.stringify({ content }),
    jsonHeaders(token)
  );
}

// List messages — POST /channels/{id}/messages/list
export function listMessages(token, channelId, limit = 50) {
  return http.post(
    `${API}/channels/${channelId}/messages/list`,
    JSON.stringify({ limit }),
    jsonHeaders(token)
  );
}

// Add reaction — POST /messages/{id}/reactions/add
export function addReaction(token, messageId, emoji) {
  return http.post(
    `${API}/messages/${messageId}/reactions/add`,
    JSON.stringify({ emoji }),
    jsonHeaders(token)
  );
}

// Typing indicator
export function startTyping(token, workspaceId, channelId) {
  return http.post(
    `${API}/workspaces/${workspaceId}/typing/start`,
    JSON.stringify({ channel_id: channelId }),
    jsonHeaders(token)
  );
}

// Search messages — POST /workspaces/{wid}/messages/search
export function searchMessages(token, workspaceId, query) {
  return http.post(
    `${API}/workspaces/${workspaceId}/messages/search`,
    JSON.stringify({ query }),
    jsonHeaders(token)
  );
}

// Standard thresholds used across tests
export const STANDARD_THRESHOLDS = {
  http_req_failed: ["rate<0.01"], // <1% errors
  http_req_duration: ["p(95)<500", "p(99)<1000"], // p95 < 500ms, p99 < 1s
};

// Stricter thresholds for read-heavy endpoints
export const READ_THRESHOLDS = {
  http_req_failed: ["rate<0.01"],
  http_req_duration: ["p(95)<300", "p(99)<500"],
};
