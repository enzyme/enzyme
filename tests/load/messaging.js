// Load test: Messaging endpoints (send, list, react, search)
//
// Tests SQLite single-connection behavior under concurrent write load.
// Logs in once during setup() to avoid rate limiting.
//
// Usage:
//   k6 run tests/load/messaging.js
//   k6 run tests/load/messaging.js --env K6_BASE_URL=https://chat.enzyme.im

import { check, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";
import {
  loginAllUsers,
  pickUser,
  sendMessage,
  listMessages,
  addReaction,
  searchMessages,
  startTyping,
  STANDARD_THRESHOLDS,
} from "./helpers.js";

// Custom metrics
const sendDuration = new Trend("msg_send_duration", true);
const listDuration = new Trend("msg_list_duration", true);
const searchDuration = new Trend("msg_search_duration", true);
const sendFailures = new Counter("msg_send_failures");

export const options = {
  scenarios: {
    // High-volume message sending (write-heavy — stresses SQLite single conn)
    message_sending: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 10 },
        { duration: "30s", target: 20 }, // 20 concurrent writers
        { duration: "15s", target: 30 }, // push the limit
        { duration: "10s", target: 0 },
      ],
      exec: "sendMessages",
    },
    // Message listing (read-heavy)
    message_reading: {
      executor: "constant-vus",
      vus: 15,
      duration: "50s",
      exec: "readMessages",
      startTime: "5s",
    },
    // Search load
    search_load: {
      executor: "constant-vus",
      vus: 5,
      duration: "40s",
      exec: "searchLoad",
      startTime: "10s",
    },
  },
  thresholds: {
    ...STANDARD_THRESHOLDS,
    msg_send_duration: ["p(95)<800", "p(99)<1500"],
    msg_list_duration: ["p(95)<300", "p(99)<500"],
    msg_search_duration: ["p(95)<1000"],
    msg_send_failures: ["count<20"],
  },
};

export function setup() {
  return loginAllUsers();
}

export function sendMessages(data) {
  const user = pickUser(data);
  const channelId = user.channels[0];
  if (!channelId) {
    sleep(2);
    return;
  }

  startTyping(user.token, user.workspaceId, channelId);

  const content = `Load test message from VU ${__VU} iter ${__ITER} at ${new Date().toISOString()}`;
  const start = Date.now();
  const res = sendMessage(user.token, channelId, content);
  sendDuration.add(Date.now() - start);

  const ok = check(res, {
    "send message status 200": (r) => r.status === 200,
    "send message has id": (r) => r.json().message?.id != null,
  });

  if (!ok) {
    sendFailures.add(1);
    sleep(1);
    return;
  }

  // Sometimes add a reaction to the message we just sent
  if (Math.random() < 0.3) {
    const msgId = res.json().message.id;
    const emojis = ["+1", "heart", "rocket", "eyes", "fire"];
    const emoji = emojis[Math.floor(Math.random() * emojis.length)];
    const reactRes = addReaction(user.token, msgId, emoji);
    check(reactRes, {
      "add reaction status 200": (r) => r.status === 200,
    });
  }

  sleep(0.5 + Math.random());
}

export function readMessages(data) {
  const user = pickUser(data);
  const channelId = user.channels[0];
  if (!channelId) {
    sleep(2);
    return;
  }

  const start = Date.now();
  const res = listMessages(user.token, channelId, 50);
  listDuration.add(Date.now() - start);

  check(res, {
    "list messages status 200": (r) => r.status === 200,
    "list messages returns array": (r) => Array.isArray(r.json().messages),
  });

  sleep(1 + Math.random());
}

export function searchLoad(data) {
  const user = pickUser(data);

  const queries = ["hello", "test", "meeting", "update", "load test", "hey"];
  const query = queries[Math.floor(Math.random() * queries.length)];

  const start = Date.now();
  const res = searchMessages(user.token, user.workspaceId, query);
  searchDuration.add(Date.now() - start);

  check(res, {
    "search status 200": (r) => r.status === 200,
  });

  sleep(2 + Math.random());
}
