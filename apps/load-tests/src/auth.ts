// Load test: Authentication endpoints (login + register)
//
// Usage:
//   k6 run apps/load-tests/dist/auth.js

import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import type { MeResponse } from './helpers.js';
import {
  SEED_USERS,
  login,
  registerUser,
  getMe,
  pickRandom,
  jsonAs,
  STANDARD_THRESHOLDS,
} from './helpers.js';

const loginDuration = new Trend('login_duration', true);
const registerDuration = new Trend('register_duration', true);
const loginFailures = new Counter('login_failures');
const registerFailures = new Counter('register_failures');

export const options = {
  scenarios: {
    login_load: {
      executor: 'ramping-vus' as const,
      startVUs: 0,
      stages: [
        { duration: '15s', target: 20 },
        { duration: '30s', target: 20 },
        { duration: '10s', target: 50 },
        { duration: '15s', target: 0 },
      ],
      exec: 'loginScenario',
    },
    register_burst: {
      executor: 'ramping-vus' as const,
      startVUs: 0,
      stages: [
        { duration: '10s', target: 5 },
        { duration: '30s', target: 10 },
        { duration: '10s', target: 0 },
      ],
      exec: 'registerScenario',
      startTime: '5s',
    },
  },
  thresholds: {
    ...STANDARD_THRESHOLDS,
    login_duration: ['p(95)<400'],
    register_duration: ['p(95)<600'],
    login_failures: ['count<10'],
    register_failures: ['count<5'],
  },
};

export function loginScenario() {
  const user = pickRandom(SEED_USERS);

  const start = Date.now();
  const token = login(user.email);
  loginDuration.add(Date.now() - start);

  if (!token) {
    loginFailures.add(1);
    sleep(1);
    return;
  }

  check(token, {
    'login returned token': (t) => t.length > 0,
  });

  const meRes = getMe(token);
  check(meRes, {
    'GET /auth/me status 200': (r) => r.status === 200,
    'GET /auth/me returns email': (r) => jsonAs<MeResponse>(r.json()).user?.email === user.email,
  });

  sleep(0.5 + Math.random());
}

export function registerScenario() {
  const start = Date.now();
  const { token, email } = registerUser(`vu${__VU}-${__ITER}`);
  registerDuration.add(Date.now() - start);

  if (!token) {
    registerFailures.add(1);
    sleep(1);
    return;
  }

  check(token, {
    'register returned token': (t) => t.length > 0,
  });

  const meRes = getMe(token);
  check(meRes, {
    'new user GET /auth/me status 200': (r) => r.status === 200,
    'new user email matches': (r) => jsonAs<MeResponse>(r.json()).user?.email === email,
  });

  sleep(1 + Math.random());
}
