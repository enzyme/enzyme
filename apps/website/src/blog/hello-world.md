---
title: 'Introducing Enzyme'
description: "Why we built a self-hostable team chat platform and what's coming next."
author: 'The Enzyme Team'
date: 2026-03-07
---

We're excited to introduce Enzyme — a self-hostable team chat platform built for teams that want to own their communication infrastructure.

## Why Another Chat Tool?

There's no shortage of team chat options. Slack, Discord, Microsoft Teams, and others all work well. But they share a common trait: your data lives on someone else's servers. For many teams, that's fine. For others, it's a dealbreaker.

We built Enzyme for that second group. Teams in regulated industries, privacy-conscious organizations, open-source communities, or simply people who prefer to own their tools.

## What Makes Enzyme Different

**Single binary, zero dependencies.** Enzyme ships as one Go binary backed by SQLite. No Docker compose files, no PostgreSQL, no Redis. Download, configure, run.

**Genuinely simple to deploy.** Our [self-hosting guide](/docs/self-hosting/) walks you through going from zero to a running instance in under five minutes. We've tested it on $5/month VPS instances — it runs comfortably.

**All the features you expect.** Real-time messaging via Server-Sent Events, threaded conversations, file uploads, emoji reactions, typing indicators, presence tracking, role-based permissions, dark mode — the things you actually use daily.

**Open source and MIT licensed.** No "open core" upsell, no enterprise features behind a paywall. The whole thing is on [GitHub](https://github.com/anthropics/enzyme).

## What's Next

We're actively building Enzyme and have a lot planned:

- **Search** — Full-text search across messages and files
- **Desktop app** — Native Electron app for macOS, Windows, and Linux
- **Notifications** — Email and push notification support
- **API** — Webhooks and a REST API for integrations

If you're interested, [try deploying it](/docs/self-hosting/) and let us know how it goes. File issues, submit pull requests, or just say hello.

We're building this in the open, and we'd love your feedback.
