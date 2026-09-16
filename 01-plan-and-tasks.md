# 01 — Plan & task list (drafted 2026-09-16, unstarted)

## 1. Problem & requirements

**Today (working but limited):** `sync-watch.sh` on the Dogebox greps
`getblockchaininfo` every 4 h and curls ntfy.sh. Limitations:

- Scoped to one script — every future service (indexer, factory, dogeboxd
  itself) would copy-paste alert logic.
- Public relay: the topic string *is* the credential (anyone who knows it can
  read/post). Already treated as semi-burned (shared with the user's devices).
- ntfy.sh is a rate-limited free public service — fine for 2 alerts/month,
  wrong for a real monitoring bus.

**Requirements:**
- R1: ntfy server runs as a Dogebox pup (NixOS container via dogeboxd), survives
  pup updates and box reboots like the other pups.
- R2: instant delivery to iOS **and** Android.
- R3: health bridge converts box events (pup state changes, node sync status,
  watchdog semantics) into topic pushes.
- R4: wow-20's existing sync-watch behavior is preserved during migration
  (no alert gap), then retired.
- R5: no new exposed attack surface beyond what's needed (LAN-first; remote
  access only via VPN, never port-forwarding).

## 2. Architecture decision

**One pup, two services** (single container, simplest that meets R1–R3):

```
┌─ ntfy-pup (NixOS container on Dogebox) ─────────────────┐
│  ntfyd (Go binary, one file)                             │
│    - listens :8099 (avoid conflicts; check dogebox map)  │
│    - auth: access tokens, per-device                     │
│    - upstream-base-url: https://ntfy.sh  ← iOS relay     │
│    - storage: /opt/dogebox/pups/storage/<pup-id>/        │
│  bridge (small script/daemon, systemd inside pup)        │
│    - polls dogeboxd pup states + node RPCs               │
│    - pushes to local ntfyd topics:                       │
│        dogebox-health, wow20-sync                        │
└──────────────────────────────────────────────────────────┘
        │ LAN (instant, Android websocket / iOS via relay)
        ▼
   phones subscribe to http://<box-LAN-IP>:8099/<topic>
   remote access: Tailscale (recommended) — not port-forwarding
```

**The iOS caveat (decides half the design):** Apple doesn't allow third-party
push servers; self-hosted ntfy gets instant iOS delivery only by relaying
"check your server" pings through ntfy.sh (`upstream-base-url`). Message
content stays on our server; ntfy.sh sees only topic hashes. Android needs no
relay (direct websocket). Alternative — pure self-hosted without relay — costs
iOS background-polling delays; not acceptable for DOWN alerts.

## 3. Task list

### Phase 0 — recon (start here)
- [ ] Read Dogebox master doc pup-authoring sections (`/Volumes/DEV Projects/DOGEBOX/docs/00-dogebox-master-context.md`)
- [ ] Study reference pup repo: `PennybagsCX/dogebox-core-txindex-pup` (manifest format, storage mapping, build flow)
- [ ] Confirm pup port allocation scheme + how dogeboxd exposes pup ports on LAN
- [ ] Confirm how pups get outbound network (bridge needs to reach ntfy.sh relay + local RPCs)

### Phase 1 — ntfy server pup
- [ ] Pup manifest + NixOS container config packaging `ntfy` (single static binary)
- [ ] `server.yml`: base-url, listen :8099, auth=true (tokens), attachment cache off, upstream-base-url for iOS
- [ ] Persistent storage dir wired through dogeboxd
- [ ] Deploy to box; create per-device access tokens
- [ ] Test: publish from box → Android instant; iPhone instant (verify relay path)
- [ ] Test: box reboot → pup returns automatically (R1)

### Phase 2 — health bridge
- [ ] Bridge v1: port sync-watch.sh semantics (RPC reachable / ibd flag / down+recover edge-detection) → topic `wow20-sync`, targeting the testnet3 node AND the mainnet txindex node
- [ ] Bridge v2: dogeboxd pup state watcher → topic `dogebox-health` (pup crashed / restarted / update-available)
- [ ] Duplicate-suppression + severity (priority) conventions documented
- [ ] Run in parallel with sync-watch.sh for ≥48 h — verify no missed/dupe alerts (R4)

### Phase 3 — cutover & wow-20 integration
- [ ] Re-point phones to self-hosted topics; retire public topic (rotate: consider it burned)
- [ ] Remove sync-watch.sh + its timer; replace with bridge (update wow-20 `08-infrastructure-status.md` §7)
- [ ] wow-20 indexer/factory/oracle: adopt ntfy publish as their alert path
- [ ] Commit + push (private repo), update docs

## 4. Open decisions (make in-session)
- Expose beyond LAN? Recommend: Tailscale on box + phones; no port-forward.
- Retention/cache window on the server (affects backfill-on-subscribe behavior seen 2026-09-16).
- One pup or two (server vs bridge separate)? Start one; split if bridge instability threatens the server.
