---
name: local-first-software
description: Principles and architecture guidance for building local-first software — apps where the primary copy of data lives on the user's device and cloud services are demoted to sync/backup. Use when designing or reviewing apps that need offline support, sync engines, CRDTs, conflict resolution, data portability, or resilience against a provider (or country) disappearing. Based on Martin Kleppmann's "Local-first in an unstable world" keynote (Local-First Conf 2025) and the 2019 Ink & Switch local-first essay.
---

# Building Local-First Software

## The core inversion

In the dominant cloud model, the server's database holds the primary copy of data and clients are thin: every read fetches from the server, every write uploads to it. Local-first inverts this relationship:

> **The copy of the data on the user's own device is the primary copy. Cloud services are demoted to a secondary role: backup and synchronization between devices and collaborators.**

Cloud services are not forbidden — they are just no longer *special*. A sync service is not the authority on what the data is. Consequences of that:

- Multiple sync services (from different providers) can be used side by side.
- Devices can sync peer-to-peer when a connection between them is possible.
- The app keeps working when the network, the provider, or the provider's *country* is unavailable.

## The seven ideals

From the 2019 Ink & Switch essay ("Local-first software: You own your data, in spite of the cloud"). Treat these as a design checklist — aspire to all, satisfy as many as feasible:

1. **No spinners** — the UI responds instantly because reads/writes hit local storage, never a network round trip.
2. **Your work is not trapped on one device** — data syncs across all the user's devices.
3. **The network is optional** — full read *and write* capability offline; sync happens whenever connectivity returns.
4. **Seamless collaboration** — real-time multi-user editing, as good as cloud apps, via sync + merge.
5. **The Long Now** — data outlives the app and the company. Open, documented formats; no forced server dependency.
6. **Security and privacy by default** — end-to-end encryption; sync servers store ciphertext they cannot read.
7. **You retain ultimate ownership and control** — the user can read, export, move, and manipulate their data without permission from anyone.

## Three levels of resilience (decide which you're targeting)

Decentralization means progressively removing single points of dependence. Each level subsumes the previous:

| Level | Dependence removed | Mechanism |
|---|---|---|
| 1 | A single **server** | Replication — copies of the data on multiple devices (classic distributed-systems fault tolerance) |
| 2 | A single **company** | Easy provider switching, ideally multiple providers used simultaneously so losing one requires no migration at all |
| 3 | A single **country** | Same as level 2, but providers deliberately chosen across jurisdictions |

The enabling primitive for levels 2–3 is **commoditization of the sync layer**: the sync service must be a standardized, interchangeable product (like wheat flour, or like IP packets) so users don't care which supplier provides it. When designing a protocol or storage format, ask: *could a competitor implement a drop-in replacement for our sync server from the docs alone?* If not, you have lock-in, not local-first.

## Architecture guidance

### Data layer
- Store the working copy in a local, embedded store (SQLite, IndexedDB/OPFS, flat files) — never treat local data as a "cache" of server truth.
- Use **CRDTs** (e.g. Automerge, Yjs, Loro) or another mergeable data model so concurrent offline edits from multiple devices converge automatically without a central arbiter. Pick the library per data model needs (rich text, JSON trees, sequences); diversity of CRDT libraries is fine — what must be standard is the sync/storage layer beneath them.
- Every mutation should be expressible as an append/merge operation, not a destructive overwrite, so history and conflict resolution stay possible.

### Sync layer
- Write the app against a **sync engine, not a REST API**. This is also the better programming model: no "did the request reach the server?" states, no timeout/retry UX, no blocking the user while a write is in flight. The client just reads and writes local data; the engine handles the distributed-systems problems.
- Keep the sync protocol dumb and generic: it moves opaque (ideally end-to-end encrypted) change sets between replicas. Business logic lives client-side.
- Support at minimum: a hosted sync server, a **self-hosted** option, and (where feasible) direct peer-to-peer sync. If the protocol is generic enough, all three are the same code path.
- Design for **multiple simultaneous sync targets** — syncing to two providers should be as natural as syncing to one.

### Portability ("incredible-journey-proofing")
Startups get acquired and shut down; the classic failure mode is "here's a zip of JSON you can't load anywhere else, you have 60 days." Make the app resilient to its own maker's death:
- The app must keep functioning locally with the vendor gone (no license phone-home, no server-gated features for core workflows).
- Data must remain syncable through *someone else's* infrastructure after the original provider disappears.
- Document the on-disk format and sync protocol well enough for independent reimplementation.
- Provide real export — complete, lossless, in the native mergeable format plus common interchange formats.

### Security
- End-to-end encrypt before sync: the server should hold ciphertext only. This is what makes "use any provider, in any country" safe rather than reckless.
- Keys belong to the user/device, with a deliberate story for key backup and multi-device key distribution.

## Honest trade-offs (design for them, don't deny them)

Centralization won for reasons; a local-first app must answer these deliberately:
- **UX polish** — centralized platforms won partly on convenience. Local-first must match cloud UX, not excuse worse UX with ideology.
- **Spam and abuse** — decentralized systems (Usenet) historically died of this. If the app has any open collaboration surface, plan moderation/abuse controls without reintroducing a central choke point.
- **Efficiency** — replication and cryptographic integrity checking cost more compute/storage than one authoritative copy. Accept and budget for it; it buys resilience.

## Why it's worth it (three tiers of motivation)

Useful for framing docs, pitches, and prioritization:
1. **User agency** — control, privacy, ownership of one's own data (the original 2019 framing).
2. **Developer pragmatism** — sync engines are a nicer programming model than REST; relying on a commodity sync backend can eliminate the need for a backend team, making software cheaper to build.
3. **Societal resilience** — software that no company or country can remotely switch off (cf. cloud lockout scenarios, the John Deere remote-disable incident) is a strategic bet for technological sovereignty.

## Anti-pattern checklist

Reject designs where:
- Local storage is merely an offline *cache* that defers to server truth on conflict.
- The client blocks user input while awaiting server acknowledgment.
- Sync only works against the vendor's hosted service, or the protocol is proprietary/undocumented.
- "Export" is lossy, partial, or requires the server to be alive.
- Conflict resolution is "last write wins" applied to whole documents (silent data loss).
- Core features require a license check or feature flag served from the vendor's infrastructure.

## Sources

- Martin Kleppmann, "Local-first in an unstable world", Local-First Conf 2025 keynote — https://youtu.be/--Ru7cL22rI
- Ink & Switch, "Local-first software: You own your data, in spite of the cloud" (2019) — https://www.inkandswitch.com/local-first/
