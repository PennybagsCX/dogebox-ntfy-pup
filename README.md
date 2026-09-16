# dogebox-ntfy-pup

**The notification pup for every Dogebox.** One install gives any box owner a
private push-notification hub: every pup, service, and script on the box can
send a notification, and any phone subscribes with one tap — no SSH, no
third-party account.

Born from wow-20's `sync-watch.sh` (hand-rolled watchdog + public ntfy topic,
pipe verified end-to-end 2026-09-16). wow-20 is the **first customer, not the
scope** — the pup is a generic platform piece any Dogebox user can install and
any pup author can build on.

**Current state:** plan only, unstarted. wow-20's stopgap (`sync-watch.sh` +
public ntfy.sh topic) stays live until this pup's cutover — **do not rip it
out until then.**

## What changes for users

| Today (wow-20's stopgap) | With the pup |
|---|---|
| SSH + curl + a public topic anyone who guesses it can read | Install pup → dashboard shows server URL + subscribe QR → scan with the ntfy app → done |
| Every script reinvents alert logic | Any pup/script: one HTTP POST = a phone notification |
| Only whatever someone remembered to wire gets alerts | Box-wide `dogebox-health` topic: pup crashes/restarts pushed automatically |
| Content transits a public relay | Self-hosted on the box; content never leaves home (iOS delivery sees only hashed topics via the ntfy.sh relay — plan §3) |

## Extensibility contract (for pup authors & scripters)

```bash
curl -d "Backup finished" \
  -H "Title: dogebox" -H "Tags: white_check_mark" \
  http://<box-LAN-IP>:8099/my-topic
```

One POST, any language, no SDK. Topics are free-form; per-device tokens gate
publish/read. Any existing ntfy integration (scripts, CI, Home Assistant, …)
works unchanged by pointing it at the box. Target UX: install from the pup
store, configure from the Dogebox dashboard (server URL, topics, tokens,
subscribe QR) — zero terminal required.

## Starting a session

```bash
cd "/Volumes/DEV Projects/dogebox-ntfy-pup"
claude
```

Read `01-plan-and-tasks.md` — requirements (incl. the user-friendly + extendable
mandate), architecture, iOS delivery caveat, phased task list.
Dogebox infrastructure background: `/Volumes/DEV Projects/DOGEBOX/docs/00-dogebox-master-context.md`;
reference pup implementation: `github.com/PennybagsCX/dogebox-core-txindex-pup`.
