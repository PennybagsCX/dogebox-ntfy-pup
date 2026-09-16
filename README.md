# ntfy pup for Dogebox

**Private push notifications for your whole box.** Install the pup, scan the QR
with the ntfy app on your phone, done — every pup crash, recovery and update
now arrives as a push notification. Anything else on the box (scripts, cron,
Home Assistant, CI) can send notifications with one HTTP POST.

Self-hosted: messages stay on your box. Token auth is on by default, nothing
is public, no third-party account. iOS + Android both get instant delivery.

## Install

1. In the Dogebox dashboard, add this repo as a **Pup Source** and install
   **ntfy** from it.
2. Click **Launch web** on the pup — its onboarding page opens.
3. Follow the three steps there: install the ntfy app
   ([Android](https://play.google.com/store/apps/details?id=io.heckel.ntfy) ·
   [iPhone](https://apps.apple.com/app/ntfy/id1625396347)), scan the QR,
   paste the device token when the app asks. Send the test notification.

That's the whole setup. No terminal.

## Send a notification from anything

```bash
curl -d "Backup finished" \
  -H "Title: dogebox" -H "Tags: white_check_mark" \
  -H "Authorization: Bearer <device token>" \
  http://<box-LAN-IP>:8099/my-topic
```

One POST, any language, no SDK. Topics are free-form (`/my-topic`, anything);
the device token from the onboarding page works everywhere. Any existing ntfy
integration works unchanged by pointing it at the box.

## The health bridge (automatic)

The pup ships with a bridge that watches **every pup on the box** via dogeboxd
and pushes to the built-in `dogebox-health` topic:

| Event | Notification |
|---|---|
| A pup crashed | 💀 `<pup> crashed` (high priority) |
| A pup recovered | ✅ `<pup> recovered` |
| A pup started / stopped | 🚀 / ⛔ `<pup> started` / `stopped` |
| Update available | 📦 `Update available: <pup>` |

Transition-only: no spam while things are healthy, duplicates suppressed
within 5 minutes, and the very first sighting after install is baseline (no
alert storm). State survives restarts.

One limit worth knowing: the server keeps messages for **12 hours**. Alerts
always arrive in real time while a phone is connected — but if a phone stays
offline longer than that, older unread alerts have already expired from the
cache. There is no cross-device history beyond 12h.

The bridge reads dogeboxd's pup list tokenless where the dashboard allows it.
If your box requires auth, the onboarding page has a one-time paste field for
a Dogebox API token — the bridge picks it up within a minute.

## Security model

- **LAN-first.** The server listens on your home network only — never
  port-forward it. For remote access use Tailscale or similar.
- **Auth on, deny-all.** Publishing and subscribing require the device token
  generated at first boot (kept in `/storage/state/`). The onboarding page
  shows credentials until you tap **Hide credentials** — the trust model is
  your own LAN, same as the dashboard.
- **iOS delivery** uses ntfy's official relay: your self-hosted server only
  ever sends ntfy.sh *hashed topic names* (never message content) to trigger
  instant delivery. The relay can only *wake* the phone — the phone still
  fetches the message from your box, so instant delivery needs both your box
  to reach ntfy.sh outbound (it does) and the phone to reach the box (your
  LAN, or VPN/Tailscale off-site). Messages sent while a phone can't reach
  the box wait in the 12-hour cache for its return. Android devices running
  the ntfy app talk to the box directly, so they work anywhere the phone can
  reach it.

## Upgrading

Installing a new version creates a fresh pup with an empty `/storage`. To keep
your phones paired and history, copy the old storage over before first boot of
the new one (from the box's shell):

```bash
sudo rsync -a /opt/dogebox/pups/storage/<old-id>/ /opt/dogebox/pups/storage/<new-id>/
```

Everything lives in that one directory: auth database, tokens, message cache.

## Building it yourself

```
dogebox.json          pup-source index → ntfy/
ntfy/
  manifest.json       pup manifest (services, exposed port 8099, metrics)
  pup.nix             Nix build: ntfy 2.28.0 (official static binary) +
                      frontdoor + bridge, both stdlib-only Go
  frontdoor/          :8099 entry — onboarding page, QR, health check,
                      reverse-proxies everything else to ntfy (127.0.0.1:8098)
  bridge/             polls dogeboxd → pushes pup-state transitions
```

One container, three services, one exposed port (`8099`). Standard library
only in both Go programs; no secrets ever baked into the image — all
credentials are generated on the box into `/storage` on first boot.

MIT licensed. Not affiliated with the Dogecoin Foundation; `ntfy` is
[binwiederhier/ntfy](https://github.com/binwiederhier/ntfy) (Apache-2.0).
