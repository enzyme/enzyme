// Load test: SSE connections (concurrent subscribers + event delivery)
//
// Lighter-weight SSE test — good for CI. For stress testing with
// thousands of connections, use sse-stress.ts instead.
//
// Usage:
//   k6 run apps/load-tests/dist/sse.js

import sse from 'k6/x/sse';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import type { UserContext } from './helpers.js';
import { BASE_URL, loginAllUsers, pickUser, sendMessage, STANDARD_THRESHOLDS } from './helpers.js';

const sseEventsReceived = new Counter('sse_events_received');
const sseConnectionErrors = new Counter('sse_connection_errors');
const eventTriggerDuration = new Trend('event_trigger_duration', true);

export const options = {
  scenarios: {
    sse_connections: {
      executor: 'ramping-vus' as const,
      startVUs: 0,
      stages: [
        { duration: '10s', target: 10 },
        { duration: '20s', target: 25 },
        { duration: '20s', target: 50 },
        { duration: '20s', target: 100 },
        { duration: '10s', target: 0 },
      ],
      exec: 'sseConnection',
    },
    event_generator: {
      executor: 'constant-arrival-rate' as const,
      rate: 5,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 10,
      exec: 'generateEvents',
      startTime: '10s',
    },
  },
  thresholds: {
    ...STANDARD_THRESHOLDS,
    sse_connection_errors: ['count<10'],
  },
};

export function setup() {
  return loginAllUsers();
}

export function sseConnection(data: UserContext[]) {
  const user = pickUser(data);
  const url = `${BASE_URL}/api/workspaces/${user.workspaceId}/events`;

  const res = sse.open(
    url,
    {
      headers: { Authorization: `Bearer ${user.token}` },
    },
    (client) => {
      client.on('event', () => {
        sseEventsReceived.add(1);
      });

      client.on('error', () => {
        sseConnectionErrors.add(1);
      });
    },
  );

  check(res, {
    'SSE connection established': (r) => r.status === 200,
  });

  sleep(1 + Math.random() * 2);
}

export function generateEvents(data: UserContext[]) {
  const user = pickUser(data);
  const channelId = user.channels[0];
  if (!channelId) return;

  const start = Date.now();
  const res = sendMessage(user.token, channelId, `SSE load test event ${Date.now()}`);
  eventTriggerDuration.add(Date.now() - start);

  check(res, {
    'event message sent': (r) => r.status === 200,
  });
}
