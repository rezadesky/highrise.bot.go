# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-10-26

### Added
- **Core Client & Lifecycle**: Functional options (`WithGatewayURL`, `WithHandshakeTimeout`, `WithAutoReconnect`), connection status tracking, graceful shutdown with OS signals.
- **WebSocket & Protocol Engine**: TLS 1.2+, Keepalive / Heartbeat ping-pong, exponential backoff with full random jitter, `MaxMessageSize` limits.
- **Event System**: Concurrency-safe event dispatcher, bounded channel queue, worker pool, event filtering, middleware pipeline, panic recovery, and unknown event fallback.
- **Action API**: Rate-limited chat queue with character validation, walk, teleport, sit, emote, reaction, tip, kick, ban, unban, outfit and inventory management.
- **Room State Manager**: In-memory live tracking of room users and coordinates.
- **Command Framework**: Command prefix, aliases, argument parsing, cooldowns, owner-only checks, and auto-generated help generator.
- **Security & Middlewares**: User allowlist, denylist (blacklist), and action audit logs.
- **Multibot Manager**: Manage and coordinate multiple Highrise bots in a single Go process.
- **Scheduler**: Periodic background task runner with interval cancellation.
- **Plugin System**: Modular plugin loader interface.
- **Storage Adapter**: In-memory store with TTL and atomic persistent JSON file storage.
- **Web API REST Client**: Dedicated endpoints for `/users`, `/rooms`, and `/items` with cursor pagination.
