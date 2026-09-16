# Social drafts — ntfy pup

> **DRAFTS ONLY — never post without explicit per-target approval.**
> All copy verified against the shipped pup v0.0.1 (2026-09-16). No claims beyond what the pup does.

---

## X / Twitter (thread, 3 posts)

**1/**
Your Dogebox runs your node. Now it can also run your notifications.

ntfy pup: your own ntfy server in a Dogebox container. Any pup, script, or service on the box can ping your phone — one HTTP POST, done.

Repo: https://github.com/PennybagsCX/dogebox-ntfy-pup

**2/**
Setup is exactly three steps:

1. Install the pup from the pup store
2. Scan the QR with the ntfy app
3. Send a test notification from the onboarding page

No terminal. No config files. No account anywhere.

**3/**
Private by default: token auth with deny-all, so only devices holding your device token can read or publish.
LAN-first (no port forwarding needed). iOS gets instant delivery through ntfy's official relay — only a hashed topic name leaves your box, never message contents.

Pup health alerts (crashed / recovered / update available) arrive automatically via the built-in health bridge.

`1 USD in DOGE tips welcome Ð`  ← drop this line before posting or keep, user call.

---

## r/dogecoin post (title + body)

**Title:** I built a notification pup for Dogebox — your node texts you when a pup crashes

**Body:**

I keep Dogebox (the Dogecoin full-node distro) running on a box at home, and I wanted my phone to tell me when something went wrong. So I built a pup for it.

**What it is:** a self-hosted ntfy notification server packaged as a Dogebox pup. After install you get an onboarding page: scan one QR with the free ntfy app (iOS/Android) and you're done. No terminal, no config.

**What you get automatically:**
- pup crashed / recovered / update-available alerts for every pup on the box (built-in health bridge)
- anything else on the box can notify you with one HTTP POST (curl example in the README)

**Privacy / security model:**
- token auth, deny-all by default — only token-holding devices can publish or subscribe
- LAN-first: no port forwarding; stay on Wi-Fi or use Tailscale for away-from-home
- iOS instant delivery relays through ntfy.sh — only a hashed topic name leaves your network, message bodies never do

**Requirements:** a Dogebox, the ntfy app (free, open source), ~5 minutes.

Repo: https://github.com/PennybagsCX/dogebox-ntfy-pup

---

## Discord blurb (Dogebox / Dogecoin dev channels, short form)

Built a pup for Dogebox: self-hosted **ntfy** push notifications. Install → scan the QR → your phone gets an alert whenever any pup crashes, recovers, or has an update. Anything on the box can also ping you with one HTTP POST (scripts, cron, Home Assistant, anything that speaks curl).

Token auth on by default, LAN-first, iOS instant delivery via ntfy's official relay (only hashed topic names leave the box).

Repo + install (add it as a Pup Source): https://github.com/PennybagsCX/dogebox-ntfy-pup

Feedback welcome — especially iOS delivery reports on your box.

---

## Upstream listing draft (Phase D — approval-gated, do not submit)

**Target:** the Dogebox community pup-source index / `Dogebox-WG` pups listing (exact venue to be confirmed with the Dogebox maintainers before anything is submitted).

**Proposed entry:**

```json
{
  "id": "pennybags.pups.ntfy",
  "name": "ntfy",
  "description": "Self-hosted push notifications for your whole box. Install, scan the QR with the ntfy app, done — pup crashes, recoveries and updates arrive as push notifications, and anything on the box can send one with a single HTTP POST. Token auth on by default, LAN-first, iOS + Android instant delivery.",
  "source": "https://github.com/PennybagsCX/dogebox-ntfy-pup.git",
  "version": "0.0.1"
}
```

**Accompanying PR/issue text:**

Adds an ntfy notification pup (v0.0.1). It packages ntfy 2.28.0 (official static binary, pinned hashes) with two small stdlib-only Go services: a frontdoor that serves a three-step onboarding page (QR pairing, device token, test button) on port 8099 and reverse-proxies the server, and a health bridge that turns dogeboxd pup-state transitions into notifications on a built-in `dogebox-health` topic (transition-only, 5-minute duplicate suppression, survives restarts).

Security posture: ntfy auth on with deny-all default; device token generated on-box at first boot (nothing baked into the image); onboarding credentials can be hidden after pairing; page is gated to LAN/private addresses. iOS instant delivery uses ntfy's official upstream relay — only SHA-256 hashed topic names are ever sent off-box.

Validation: 24-check e2e battery (publish/subscribe through the frontdoor proxy, deny-all 403s, transition + dup-suppression behavior, token save/clear, CSRF and DNS-rebind gates, hide/reveal), go vet + linux/amd64 cross-build clean, shellcheck clean on the boot script.

Happy to iterate on manifest conventions or packaging style to match house patterns.
