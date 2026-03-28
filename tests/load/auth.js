// Load test: Authentication endpoints (login + register)
//
// Usage:
//   k6 run tests/load/auth.js
//   k6 run tests/load/auth.js --env K6_BASE_URL=https://chat.enzyme.im

import { check, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";
import {
  SEED_USERS,
  login,
  registerUser,
  getMe,
  STANDARD_THRESHOLDS,
} from "./helpers.js";

// Custom metrics
const loginDuration = new Trend("login_duration", true);
const registerDuration = new Trend("register_duration", true);
const loginFailures = new Counter("login_failures");
const registerFailures = new Counter("register_failures");

export const options = {
  scenarios: {
    // Sustained login load — simulates many users logging in
    login_load: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "15s", target: 20 }, // ramp up
        { duration: "30s", target: 20 }, // steady state
        { duration: "10s", target: 50 }, // spike
        { duration: "15s", target: 0 }, // ramp down
      ],
      exec: "loginScenario",
    },
    // Registration burst — new users signing up
    register_burst: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 5 },
        { duration: "30s", target: 10 },
        { duration: "10s", target: 0 },
      ],
      exec: "registerScenario",
      startTime: "5s", // stagger start
    },
  },
  thresholds: {
    ...STANDARD_THRESHOLDS,
    login_duration: ["p(95)<400"],
    register_duration: ["p(95)<600"],
    login_failures: ["count<10"],
    register_failures: ["count<5"],
  },
};

// Note: This test intentionally calls login per-iteration to stress the auth
// endpoint. Rate limiting may cause failures at high VU counts — that's part
// of what we're measuring.

export function loginScenario() {
  const user = SEED_USERS[Math.floor(Math.random() * SEED_USERS.length)];

  const start = Date.now();
  const token = login(user.email);
  loginDuration.add(Date.now() - start);

  const loginOk = check(token, {
    "login returned token": (t) => t !== null && t.length > 0,
  });

  if (!loginOk) {
    loginFailures.add(1);
    sleep(1);
    return;
  }

  // Verify token works with /auth/me
  const meRes = getMe(token);
  check(meRes, {
    "GET /auth/me status 200": (r) => r.status === 200,
    "GET /auth/me returns email": (r) => r.json().user?.email === user.email,
  });

  sleep(0.5 + Math.random());
}

export function registerScenario() {
  const start = Date.now();
  const { token, email } = registerUser(`vu${__VU}-${__ITER}`);
  registerDuration.add(Date.now() - start);

  const regOk = check(token, {
    "register returned token": (t) => t !== null && t.length > 0,
  });

  if (!regOk) {
    registerFailures.add(1);
    sleep(1);
    return;
  }

  // Verify the new account works
  const meRes = getMe(token);
  check(meRes, {
    "new user GET /auth/me status 200": (r) => r.status === 200,
    "new user email matches": (r) => r.json().user?.email === email,
  });

  sleep(1 + Math.random());
}
