// SSE stress test — holds many concurrent SSE connections while generating
// realistic traffic (messages, reactions, typing) and measuring end-to-end latency.
//
// Usage:
//   k6 run apps/load-tests/dist/sse-stress.js
//   k6 run apps/load-tests/dist/sse-stress.js --env SSE_CONNECTIONS=2000 --env SSE_DURATION=2m

import sse from 'k6/x/sse';
import { check } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import type { UserContext, SendMessageResponse } from './helpers.js';
import {
  BASE_URL,
  loginAllUsers,
  pickUser,
  pickRandom,
  jsonAs,
  sendMessage,
  addReaction,
  startTyping,
  REACTION_EMOJIS,
} from './helpers.js';

const e2eLatency = new Trend('sse_e2e_latency', true);
const sseEventsReceived = new Counter('sse_events_received');
const sseConnectionErrors = new Counter('sse_connection_errors');
const sseMsgSent = new Counter('sse_messages_sent');
const sseMsgErrors = new Counter('sse_message_errors');

// Configuration via env vars (with upper bounds to prevent accidental self-DoS)
const CONNECTIONS = parseInt(__ENV.SSE_CONNECTIONS || '100', 10);
const DURATION = __ENV.SSE_DURATION || '2m';
const MSG_RATE = parseInt(__ENV.SSE_MSG_RATE || '5', 10);
const RAMP = __ENV.SSE_RAMP || '30s';

export const options = {
  scenarios: {
    // Each VU holds one SSE connection for the duration of the test.
    sse_listeners: {
      executor: 'ramping-vus' as const,
      startVUs: 0,
      stages: [
        { duration: RAMP, target: CONNECTIONS },
        { duration: DURATION, target: CONNECTIONS },
        { duration: '10s', target: 0 },
      ],
      exec: 'holdConnection',
      gracefulStop: '10s',
    },
    // Sends messages at a fixed rate while SSE connections are open.
    activity: {
      executor: 'constant-arrival-rate' as const,
      rate: MSG_RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 10,
      maxVUs: 20,
      exec: 'generateActivity',
      startTime: RAMP,
    },
  },
  thresholds: {
    sse_connection_errors: ['count<50'],
    sse_message_errors: ['count<10'],
  },
};

export function setup() {
  return loginAllUsers();
}

export function holdConnection(data: UserContext[]) {
  const user = pickUser(data);
  const url = `${BASE_URL}/api/workspaces/${user.workspaceId}/events`;

  const res = sse.open(
    url,
    {
      headers: { Authorization: `Bearer ${user.token}` },
    },
    (client) => {
      client.on('event', (event) => {
        sseEventsReceived.add(1);

        // Measure end-to-end latency for messages with embedded timestamps.
        // Senders embed "t=<millis>" in message content. This measures
        // send RTT + server broadcast + SSE delivery, not just fan-out.
        if (event.name === 'message_created' && event.data?.includes('t=')) {
          const match = event.data.match(/t=(\d+)/);
          if (match) {
            const sentMs = parseInt(match[1], 10);
            const latencyMs = Date.now() - sentMs;
            if (latencyMs > 0 && latencyMs < 30000) {
              e2eLatency.add(latencyMs);
            }
          }
        }
      });

      client.on('error', () => {
        sseConnectionErrors.add(1);
      });
    },
  );

  check(res, {
    'SSE connection established': (r) => r.status === 200,
  });
}

export function generateActivity(data: UserContext[]) {
  const user = pickUser(data);
  if (user.channels.length === 0) return;

  const channelId = pickRandom(user.channels);

  const res = sendMessage(user.token, channelId, `t=${Date.now()} Load test from ${user.email}`);

  if (res.status === 200) {
    sseMsgSent.add(1);

    if (Math.random() < 0.3) {
      try {
        const msgId = jsonAs<SendMessageResponse>(res.json()).message.id;
        if (msgId) {
          addReaction(user.token, msgId, pickRandom(REACTION_EMOJIS));
        }
      } catch {
        sseMsgErrors.add(1);
      }
    }
  } else {
    sseMsgErrors.add(1);
  }

  if (Math.random() < 0.2) {
    startTyping(user.token, user.workspaceId, channelId);
  }
}
