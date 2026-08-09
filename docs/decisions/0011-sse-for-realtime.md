# Use SSE for lightweight realtime

**Status:** Accepted

## Context
Notifications and timeline/task refresh are server-to-client and do not require a general messaging stack.

## Decision
Use authenticated Server-Sent Events with a bounded in-process hub.

## Consequences
The implementation is small and browser-native. V0.1 events are not shared across API replicas and clients must refetch authoritative state.
