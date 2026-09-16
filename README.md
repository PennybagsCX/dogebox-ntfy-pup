# dogebox-ntfy-pup

Self-hosted ntfy notification service for the Dogebox, born from wow-20's
`sync-watch.sh` — a hand-rolled watchdog + public ntfy topic that already
carries wow-20's sync alerts (install and end-to-end delivery verified by
test push, 2026-09-16; DOWN-alert branch dry-run-verified).

**Current state:** one hand-rolled script + systemd timer pushes to the public
ntfy.sh service (topic `wow20-tn3-k7q2xv9m`). It works and stays in place until
this project replaces it — **do not rip it out until the pup cutover is done.**

**What the pup adds:**
1. A self-hosted **ntfy server** on the box — alerts no longer depend on a
   public third-party relay, and the topic string stops being a bearer token
   readable by anyone who guesses it.
2. A **health bridge** — one place that watches dogeboxd pup states + node
   sync states and pushes alerts, instead of every script re-implementing
   grep-and-curl logic.
3. A **notification bus for wow-20** — indexer/factory/oracle events (and
   later mainnet monitoring) flow through the same pipe.

## Starting a session

```bash
cd "/Volumes/DEV Projects/dogebox-ntfy-pup"
claude
```

Read `01-plan-and-tasks.md` — it has the requirements, architecture decision,
iOS delivery caveat, and the phased task list to build this end-to-end.
Dogebox infrastructure background: `/Volumes/DEV Projects/DOGEBOX/docs/00-dogebox-master-context.md`;
reference pup implementation: `github.com/PennybagsCX/dogebox-core-txindex-pup`.
