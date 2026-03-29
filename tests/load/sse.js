// Load test: SSE connections (concurrent subscribers + event delivery)
//
// Tests maximum concurrent SSE connections and memory behavior.
// SSE connections are long-lived, so this test holds connections open
// while also generating events to measure delivery.
//
// Usage:
//   k6 run tests/load/sse.js
//   k6 run tests/load/sse.js --env K6_BASE_URL=https://chat.enzyme.im

import { check, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";
import http from "k6/http";
import {
  BASE_URL,
  loginAllUsers,
  pickUser,
  sendMessage,
  STANDARD_THRESHOLDS,
} from "./helpers.js";

// Custom metrics
const sseConnectDuration = new Trend("sse_connect_duration", true);
const sseConnections = new Counter("sse_connections_opened");
const sseFailures = new Counter("sse_connection_failures");
const eventTriggerDuration = new Trend("event_trigger_duration", true);

export const options = {
  scenarios: {
    // Ramp up SSE connections to test concurrency limits
    sse_connections: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 10 }, // 10 concurrent SSE connections
        { duration: "20s", target: 25 }, // 25 concurrent
        { duration: "20s", target: 50 }, // push to 50
        { duration: "20s", target: 100 }, // stress test at 100
        { duration: "10s", target: 0 }, // ramp down
      ],
      exec: "sseConnection",
    },
    // Generate events while SSE connections are open
    event_generator: {
      executor: "constant-arrival-rate",
      rate: 5, // 5 messages per second
      timeUnit: "1s",
      duration: "60s",
      preAllocatedVUs: 10,
      exec: "generateEvents",
      startTime: "10s", // start after some SSE connections are up
    },
  },
  thresholds: {
    ...STANDARD_THRESHOLDS,
    sse_connect_duration: ["p(95)<1000"],
    sse_connection_failures: ["count<10"],
  },
};

export function setup() {
  return loginAllUsers();
}

export function sseConnection(data) {
  const user = pickUser(data);

  const sseUrl = `${BASE_URL}/api/workspaces/${user.workspaceId}/events`;
  const start = Date.now();

  const res = http.get(sseUrl, {
    headers: {
      Authorization: `Bearer ${user.token}`,
      Accept: "text/event-stream",
      "Cache-Control": "no-cache",
    },
    timeout: "15s",
  });

  sseConnectDuration.add(Date.now() - start);
  sseConnections.add(1);

  const ok = check(res, {
    "SSE connection established": (r) => r.status === 200,
    "SSE content type": (r) =>
      r.headers["Content-Type"]?.includes("text/event-stream") ?? false,
    "SSE body has events": (r) => r.body?.includes("event:") ?? false,
  });

  if (!ok) {
    sseFailures.add(1);
  }

  sleep(1 + Math.random() * 2);
}

export function generateEvents(data) {
  const user = pickUser(data);
  const channelId = user.channels[0];
  if (!channelId) return;

  const start = Date.now();
  const res = sendMessage(
    user.token,
    channelId,
    `SSE load test event ${Date.now()}`
  );
  eventTriggerDuration.add(Date.now() - start);

  check(res, {
    "event message sent": (r) => r.status === 200,
  });
}
