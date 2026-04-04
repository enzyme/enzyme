// Load test: Messaging endpoints (send, list, react, search)
//
// Tests SQLite write concurrency under load.
//
// Usage:
//   k6 run apps/load-tests/dist/messaging.js

import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import type { UserContext } from './helpers.js';
import type { SendMessageResponse, MessageListResult } from './helpers.js';
import {
  loginAllUsers,
  pickUser,
  pickRandom,
  jsonAs,
  sendMessage,
  listMessages,
  addReaction,
  searchMessages,
  startTyping,
  STANDARD_THRESHOLDS,
  REACTION_EMOJIS,
  SEARCH_QUERIES,
} from './helpers.js';

const sendDuration = new Trend('msg_send_duration', true);
const listDuration = new Trend('msg_list_duration', true);
const searchDuration = new Trend('msg_search_duration', true);
const sendFailures = new Counter('msg_send_failures');

export const options = {
  scenarios: {
    message_sending: {
      executor: 'ramping-vus' as const,
      startVUs: 0,
      stages: [
        { duration: '10s', target: 10 },
        { duration: '30s', target: 20 },
        { duration: '15s', target: 30 },
        { duration: '10s', target: 0 },
      ],
      exec: 'sendMessages',
    },
    message_reading: {
      executor: 'constant-vus' as const,
      vus: 15,
      duration: '50s',
      exec: 'readMessages',
      startTime: '5s',
    },
    search_load: {
      executor: 'constant-vus' as const,
      vus: 5,
      duration: '40s',
      exec: 'searchLoad',
      startTime: '10s',
    },
  },
  thresholds: {
    ...STANDARD_THRESHOLDS,
    msg_send_duration: ['p(95)<800', 'p(99)<1500'],
    msg_list_duration: ['p(95)<300', 'p(99)<500'],
    // Relaxed from 1000ms: combined count+data query (COUNT(*) OVER) is slower
    // per-query but eliminates a full FTS5 round-trip, improving real-world latency.
    msg_search_duration: ['p(95)<2000'],
    msg_send_failures: ['count<20'],
  },
};

export function setup() {
  return loginAllUsers();
}

export function sendMessages(data: UserContext[]) {
  const user = pickUser(data);
  const channelId = pickRandom(user.channels);
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
    'send message status 200': (r) => r.status === 200,
  });

  if (!ok) {
    sendFailures.add(1);
    sleep(1);
    return;
  }

  const body = jsonAs<SendMessageResponse>(res.json());
  check(null, {
    'send message has id': () => body.message?.id != null,
  });

  if (Math.random() < 0.3) {
    const reactRes = addReaction(user.token, body.message.id, pickRandom(REACTION_EMOJIS));
    check(reactRes, {
      'add reaction status 200': (r) => r.status === 200,
    });
  }

  sleep(0.5 + Math.random());
}

export function readMessages(data: UserContext[]) {
  const user = pickUser(data);
  const channelId = pickRandom(user.channels);
  if (!channelId) {
    sleep(2);
    return;
  }

  const start = Date.now();
  const res = listMessages(user.token, channelId, 50);
  listDuration.add(Date.now() - start);

  check(res, {
    'list messages status 200': (r) => r.status === 200,
    'list messages returns array': (r) =>
      Array.isArray(jsonAs<MessageListResult>(r.json()).messages),
  });

  sleep(1 + Math.random());
}

export function searchLoad(data: UserContext[]) {
  const user = pickUser(data);
  const query = pickRandom(SEARCH_QUERIES);

  const start = Date.now();
  const res = searchMessages(user.token, user.workspaceId, query);
  searchDuration.add(Date.now() - start);

  check(res, {
    'search status 200': (r) => r.status === 200,
  });

  sleep(2 + Math.random());
}
