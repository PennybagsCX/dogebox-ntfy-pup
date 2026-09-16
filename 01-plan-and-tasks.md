# 01 — Plan & task list (drafted 2026-09-16, unstarted)

> **Product mandate (user, 2026-09-16):** "something super user friendly,
> simple, and extendable" — usable for *anything/everything* by *any* Dogebox
> owner, not a wow-20-internal tool. Every design choice below serves that.

## 1. Problem & requirements

**Today (works, but developer-grade only):** wow-20's `sync-watch.sh` greps
`getblockchaininfo` every 4 h and curls the public ntfy.sh service. For a
regular Dogebox owner that path is a non-starter: it needs SSH, hand-written
curl, and a topic string that doubles as a world-readable password.

**Requirements:**
- R1: ntfy server runs as a Dogebox pup (NixOS container via dogeboxd), survives
  pup updates and box reboots like the other pups.
- R2: instant delivery to iOS **and** Android.
- R3: health bridge converts box events into topic pushes — **generic by
  default**: all pup state changes box-wide (`dogebox-health`). **No
  project-specific topics ship with the pup.**
- R4: wow-20's existing sync-watch behavior is preserved during migration
  (no alert gap), then retired. (Migration is a wow-20-side concern — nothing
  wow-20-specific ships in the pup.)
- R5: no new exposed attack surface (LAN-first; remote access only via VPN,
  never port-forwarding); auth tokens on by default.
- **R6 (user-friendly):** a non-technical owner can go install → phone
  receiving alerts with no terminal: pup-store install, dashboard config,
  QR/deep-link subscribe, auto-generated tokens.
- **R7 (extendable):** the stable integration surface for any pup author or
  script is a single HTTP POST (documented contract, README §extensibility).
  No SDK, no dogeboxd API knowledge required.

## 2. Architecture decision

**One pup, two services** (single container, simplest that meets R1–R3, R7):

```
┌─ ntfy-pup (NixOS container on Dogebox) ─────────────────┐
│  ntfyd (Go binary, one file)                             │
│    - listens :8099 (avoid conflicts; check dogebox map)  │
│    - auth: access tokens, per-device                     │
│    - upstream-base-url: https://ntfy.sh  ← iOS relay     │
│    - storage: /opt/dogebox/pups/storage/<pup-id>/        │
│  bridge (small script/daemon, systemd inside pup)        │
│    - polls dogeboxd pup states (ALL pups, generic)       │
│    - pushes to local ntfyd topic:                        │
│        dogebox-health  (auto, every pup)                 │
│                                                          │
│  Consumer-specific checks (e.g. wow-20 node sync) are    │
│  NOT part of the shipped pup — they live on the          │
│  consumer's side and POST via the one-line contract.     │
└──────────────────────────────────────────────────────────┘
        │ LAN (instant, Android websocket / iOS via relay)
        ▼
   phones subscribe via QR / ntfy:// deep link
   remote access: Tailscale (recommended) — not port-forwarding
```

## 3. The iOS caveat (decides half the design)

Apple doesn't allow third-party push servers; self-hosted ntfy gets instant
iOS delivery only by relaying "check your server" pings through ntfy.sh
(`upstream-base-url`). Message content stays on our server; ntfy.sh sees only
topic hashes. Android needs no relay (direct websocket). Pure self-hosted
without relay costs iOS background-polling delays — not acceptable for
down-alerts. This stays true for every user who installs the pup; document it
plainly in the pup's README/dashboard blurb.

## 4. UX for non-technical owners (R6 — the product bar)

- **Install:** one click from the pup store (verify the publish path in
  Phase 0 — how third-party pups reach the store/index).
- **First-run screen in the Dogebox dashboard:** server URL, one **subscribe
  QR** (ntfy apps accept `ntfy://<server>/<topic>` deep links), a default
  device token, and a live "send test notification" button.
- **Topics:** `dogebox-health` pre-created and auto-wired to the bridge;
  extra topics are free-form (type a name, get a token).
- **Defaults that are safe:** auth on, tokens required, LAN binding only.
- **Copy-paste contract** shown in-dashboard for developers (the one-liner
  curl from the README).

## 5. Task list

### Phase 0 — recon (start here)
- [ ] Read Dogebox master doc pup-authoring sections (`/Volumes/DEV Projects/DOGEBOX/docs/00-dogebox-master-context.md`)
- [ ] Study reference pup repo: `PennybagsCX/dogebox-core-txindex-pup` (manifest format, storage mapping, build flow)
- [ ] **Pup UI capability:** can a pup render a config/first-run screen in the Dogebox dashboard? (R6 hinges on this; if not, fall back to a small web UI served by the pup itself)
- [ ] **Pup-store publishing path:** how does a third-party pup become installable by other users?
- [ ] Confirm pup port allocation scheme + how dogeboxd exposes pup ports on LAN
- [ ] Confirm how pups get outbound network (bridge needs the ntfy.sh relay + local RPCs)

### Phase 1 — ntfy server pup
- [ ] Pup manifest + NixOS container config packaging `ntfy` (single static binary)
- [ ] `server.yml`: base-url, listen :8099, auth=true (tokens), attachment cache off, upstream-base-url for iOS
- [ ] Persistent storage dir wired through dogeboxd
- [ ] First-run UX: dashboard screen with server URL + subscribe QR + device token + "send test" button (see §4; fall back to pup-served web UI if no dashboard hook)
- [ ] Deploy to box; create per-device access tokens
- [ ] Test: publish from box → Android instant; iPhone instant (verify relay path)
- [ ] Test: box reboot → pup returns automatically (R1)

### Phase 2 — health bridge (generic only)
- [ ] Bridge: dogeboxd pup state watcher → topic `dogebox-health` (any pup crashed / restarted / update-available) — the box-wide feature every user gets for free
- [ ] Duplicate-suppression + severity (priority) conventions documented in the pup README
- [ ] Run the pup's health alerts in parallel with the old alert stack for ≥48 h — verify no missed/dupe alerts (R4)
- [ ] **Wow-20 side — private consumer, NOT shipped in the pup:** evolve `sync-watch.sh` into a script that POSTs its node checks (RPC reachable / ibd flag / down+recover edge-detection) to the pup's local server via the one-line contract, covering the testnet3 AND mainnet txindex nodes

### Phase 3 — cutover, wow-20 integration, publish
- [ ] Re-point phones to self-hosted topics; retire public topic (rotate: consider it burned)
- [ ] Remove sync-watch.sh + its timer; replace with bridge (update wow-20 `08-infrastructure-status.md` §7)
- [ ] wow-20 indexer/factory/oracle: adopt the one-POST contract as their alert path
- [ ] Publish the pup (pup store / index per Phase 0 findings); user-facing README: install → scan QR → done
- [ ] Commit + push (private repo during development), update docs

## 6. Open decisions (make in-session)
- Expose beyond LAN? Recommend: Tailscale on box + phones; no port-forward.
- Retention/cache window on the server (affects backfill-on-subscribe behavior seen 2026-09-16).
- One pup or two (server vs bridge separate)? Start one; split if bridge instability threatens the server.
- Default priority/notification conventions for `dogebox-health` (min priority so iOS delivers silently vs loudly?).
