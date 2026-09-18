package main

// Server-rendered onboarding pages. Placeholders are substituted with
// strings.NewReplacer in serveOnboarding/serveHidden. Kept as separate
// consts so frontdoor.go stays readable; no backticks anywhere (Go raw
// strings), no external assets except the embedded QR library at /__qr.js.
//
// Designed around field reality: iOS Safari force-upgrades http:// to
// https:// and then dies with an SSL error against this intentionally
// plain-HTTP box — so the page is written to be read on a COMPUTER screen
// and never asks anyone to open it on a phone. The phone-side flow uses
// only things that cannot hit that trap: the ntfy app itself (plain HTTP),
// the Camera app scanning a plain-TEXT key QR (Camera shows a Copy button
// for text QRs — nothing opens), and on Android the ntfy:// deep link
// (Camera hands it straight to the app, no browser).

const onboardingHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ntfy — set up notifications</title>
<style>
  :root { --gold:{{GOLD}}; --bg:#171410; --card:#201c15; --line:#3a3325; --text:#efe9dc; --dim:#a89e88; }
  * { box-sizing:border-box; }
  body { margin:0; background:var(--bg); color:var(--text); font:16px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif; }
  main { max-width:720px; margin:0 auto; padding:28px 20px 72px; }
  h1 { font-size:1.65rem; margin:14px 0 4px; }
  h1 .dot { color:var(--gold); }
  p.sub { color:var(--dim); margin:0 0 26px; }
  .card { background:var(--card); border:1px solid var(--line); border-radius:14px; padding:20px; margin-bottom:16px; }
  .step { display:flex; align-items:center; gap:10px; margin:0 0 12px; }
  .num { background:var(--gold); color:#171410; font-weight:700; min-width:26px; height:26px; border-radius:50%; display:flex; align-items:center; justify-content:center; flex:none; font-size:0.9rem; padding:0 6px; }
  h2 { font-size:1.05rem; margin:0; font-weight:600; }
  code, .mono { font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace; font-size:0.88em; word-break:break-all; }
  .row { display:flex; gap:8px; margin-top:10px; align-items:stretch; }
  .row code { flex:1; background:#14110c; border:1px solid var(--line); border-radius:8px; padding:10px 12px; display:flex; align-items:center; }
  button { background:var(--gold); border:0; color:#171410; font-weight:600; border-radius:8px; padding:10px 16px; cursor:pointer; font-size:0.95rem; }
  button:hover { filter:brightness(1.08); }
  button.ghost { background:transparent; border:1px solid var(--line); color:var(--text); }
  .qrbox { background:#fff; display:inline-block; padding:16px; border-radius:12px; margin-top:8px; }
  .qrbox svg { display:block; width:260px; height:260px; }
  .qrlabel { margin:6px 0 0; font-size:0.85rem; }
  .muted { color:var(--dim); font-size:0.92rem; }
  a { color:var(--gold); }
  input[type=password],input[type=text] { flex:1; width:100%; background:#14110c; border:1px solid var(--line); color:var(--text); border-radius:8px; padding:10px 12px; font-size:0.95rem; margin-top:10px; }
  #test-status, #bridge-status { margin-top:10px; min-height:1.4em; color:var(--dim); }
  details.card { margin-top:0; }
  summary { cursor:pointer; color:var(--text); font-weight:600; }
  summary .muted { font-weight:400; }
  pre { background:#14110c; border:1px solid var(--line); border-radius:8px; padding:12px; overflow-x:auto; font-size:0.82rem; margin:10px 0; }
  ul { margin:8px 0 0; padding-left:20px; }
  ol { margin:8px 0 0; padding-left:22px; }
  li { margin:6px 0; }
  .platform { border:1px solid var(--line); border-radius:10px; padding:12px 14px; margin-top:10px; }
  .platform h3 { margin:0 0 6px; font-size:0.95rem; }
  .note { border:1px solid var(--line); border-left:3px solid var(--gold); border-radius:10px; padding:10px 14px; margin-bottom:20px; }
  form.inline { display:inline; }
</style>
</head>
<body>
<main>
  <h1>Notifications on your phone<span class="dot">.</span></h1>
  <p class="sub">Your Dogebox can send real push notifications — when a pup crashes or recovers, when updates land, or from anything you connect. Four short steps, about 3 minutes, no terminal.</p>

  <div class="note">
    <b>Reading this on your phone?</b> If it refuses to open (an &ldquo;SSL error&rdquo;), that is your phone forcing encryption this private box intentionally doesn't speak. No problem — <b>you never need this page on your phone.</b> Open it on a computer instead; your phone only needs its ntfy app and its Camera, and everything below works from there.
  </div>

  <div class="card">
    <div class="step"><span class="num">1</span><h2>Install the ntfy app on your phone</h2></div>
    <p class="muted">ntfy is the free app that shows your box's messages as normal notifications:</p>
    <ul>
      <li><a href="https://play.google.com/store/apps/details?id=io.heckel.ntfy" target="_blank" rel="noopener">Android — get it on Google Play</a></li>
      <li><a href="https://apps.apple.com/app/ntfy/id1625396347" target="_blank" rel="noopener">iPhone — get it on the App Store</a></li>
    </ul>
    <p class="muted">Open it once after installing (tap <b>Open</b> in the store). Already have ntfy? Skip to step 2.</p>
  </div>

  <div class="card">
    <div class="step"><span class="num">2</span><h2>Put the key on your phone</h2></div>
    <p class="muted">Your box is private — only devices holding this key may connect. The app will ask for it in step 3. Two easy ways to get it there:</p>

    <div class="platform">
      <h3>✨ Easiest — same Apple ID on this computer and your iPhone?</h3>
      <ol>
        <li>Click <b>Copy</b> below (on this computer).</li>
        <li>On your iPhone, open <b>Notes</b> (or Messages, or anything with a keyboard) and <b>paste</b>. The key appears — Universal Clipboard carried it over.</li>
      </ol>
    </div>

    <div class="platform">
      <h3>📷 Works on every phone — scan the key</h3>
      <ol>
        <li>Open the <b>Camera</b> app and point it at this code.</li>
        <li>The key appears <b>as text</b> on the phone — tap <b>Copy</b> (on iPhone) or tap the text then copy (Android). Nothing opens; nothing loads. You're just grabbing text.</li>
      </ol>
      <div id="qrcode-key" class="qrbox"><noscript>Enable JavaScript to see the QR code.</noscript></div>
      <p class="muted qrlabel">↑ the key as plain text — your Camera app can copy it directly</p>
    </div>

    <div class="row"><code id="token">{{TOKEN}}</code><button class="ghost" onclick="copyText('token', this)">Copy</button></div>
    <p class="muted">The same key works on every phone in your house. (Once all your phones are connected, hide it — see the bottom of this page.)</p>
  </div>

  <div class="card">
    <div class="step"><span class="num">3</span><h2>In the app, add this server</h2></div>
    <p class="muted">These two values go into the app — the server address you'll type, the topic it fills in for you:</p>
    <div class="row"><code id="server-url">{{HOST_URL}}</code><button class="ghost" onclick="copyText('server-url', this)">Copy</button></div>
    <p class="muted">Topic: <span class="mono">{{TOPIC}}</span></p>

    <div class="platform">
      <h3>🤖 Android — fast way</h3>
      <ol>
        <li>Point the <b>Camera</b> app at this code — a link pops up.</li>
        <li><b>Tap the link</b> (not &ldquo;open in browser&rdquo;) — the ntfy app opens with everything filled in.</li>
        <li>Tap <b>Subscribe</b>. Login box: <b>Username</b> anything (e.g. <span class="mono">token</span>), <b>Password</b> — paste the key from step 2.</li>
      </ol>
      <div id="qrcode-link" class="qrbox"><noscript>Enable JavaScript to see the QR code.</noscript></div>
      <p class="muted qrlabel">↑ hands the topic straight to the ntfy app — no browser involved</p>
      <p class="muted">Link didn't open the app? Use the manual steps below — they work identically on Android.</p>
    </div>

    <div class="platform">
      <h3>📱 Manual steps — identical on iPhone and Android</h3>
      <ol>
        <li>In the ntfy app tap <b>+</b> (top right).</li>
        <li>Under <b>Topic</b> enter: <span class="mono">{{TOPIC}}</span></li>
        <li>Turn on <b>Use another server</b> (or switch off the default server) and enter: <span class="mono">{{HOST_URL}}</span></li>
        <li>Tap <b>Subscribe</b>. A login box appears: <b>Username</b> anything (e.g. <span class="mono">token</span>), <b>Password</b> — <b>paste the key</b> from step 2.</li>
        <li>No login box? Open <b>Settings → Manage users → +</b> (iPhone) or long-press the subscription → <b>Settings</b> (Android): server <span class="mono">{{HOST_URL}}</span>, choose the <b>API key / token</b> type, paste the key.</li>
      </ol>
      <p class="muted">Don't have the key on the phone yet? Go back to step 2 — it takes ten seconds.</p>
    </div>
  </div>

  <div class="card">
    <div class="step"><span class="num">4</span><h2>Test it</h2></div>
    <p class="muted">Make sure your phone shows the <span class="mono">{{TOPIC}}</span> subscription from step 3, then press the button:</p>
    <button onclick="sendTest()">Send a test notification</button>
    <div id="test-status"></div>
    <p class="muted">Your phone should buzz within a few seconds — with the app open you'll see it in the list, with the app closed you'll get a normal notification banner.</p>
  </div>

  <details class="card">
    <summary>Not working? Start here <span class="muted">(the five common problems)</span></summary>
    <ul>
      <li><b>&ldquo;An SSL error has occurred&rdquo; on the phone.</b> You tried to open this page (or the server) in a phone browser. Phones force HTTPS; this box is deliberately plain HTTP on your private network. Nothing is broken — close the browser and use the app-only steps above.</li>
      <li><b>The subscription shows a red icon or an error.</b> The key was pasted incorrectly. Android: long-press the subscription → <i>Settings</i> → re-enter the key as the password. iPhone: <i>Settings → Manage users</i> → tap your user → re-paste.</li>
      <li><b>Messages only arrive while the app is open (Android).</b> Your phone is putting ntfy to sleep. Phone <i>Settings → Apps → ntfy → Battery</i> → choose <b>Unrestricted</b>. Keep <i>Instant delivery</i> switched on for the subscription.</li>
      <li><b>Nothing arrives at all.</b> The phone must be able to reach the box: same home network, or Tailscale connected (the server address starts with <span class="mono">100.</span>? That one is Tailscale-only).</li>
      <li><b>Scanning a code did nothing.</b> Phone cameras are picky about screens. Hold steady, ~15&nbsp;cm away, brightness up. Or skip scanning entirely — everything the codes contain is also written as text on this page.</li>
    </ul>
  </details>

  <details class="card">
    <summary>Connect the health bridge <span class="muted">(usually automatic)</span></summary>
    <p class="muted">The bridge pushes pup crashes and recoveries to <span class="mono">{{TOPIC}}</span>. It reads the box on its own when allowed; if the dashboard shows the bridge idle, paste a Dogebox API token here (Dashboard → browser devtools → localStorage <span class="mono">storeState.networkContext.token</span>) and it picks it up within a minute.</p>
    <input type="password" id="bridge-input" placeholder="Dogebox API token (optional)" autocomplete="off">
    <div class="row"><button onclick="saveBridge()">Save</button><button class="ghost" onclick="clearBridge()">Clear</button></div>
    <div id="bridge-status"></div>
  </details>

  <details class="card">
    <summary>Send notifications from anything <span class="muted">(one HTTP POST)</span></summary>
    <p>Server URL: <span class="mono">{{HOST_URL}}</span></p>
    <pre>curl -d "Backup finished" \
  -H "Title: dogebox" -H "Tags: white_check_mark" \
  -H "Authorization: Bearer {{TOKEN}}" \
  {{HOST_URL}}/my-topic</pre>
    <p class="muted">Any ntfy integration works — scripts, cron, Home Assistant, Uptime Kuma, CI — just point it at this server with the key from step 2. Extra topics are free-form: send to <span class="mono">/any-topic-name</span>, then add it in the app exactly like step 3.</p>
  </details>

  <div class="card">
    <p class="muted">The key on this page stays visible until you hide it — anyone on your home network could read it, which is the trust model of a LAN-first box. Never port-forward this to the internet; use Tailscale or similar for remote access. Once every phone is connected:</p>
    <form method="post" action="/onboard/hide"><button class="ghost" type="submit">Hide the key on this page</button></form>
    <p class="muted">Hidden by mistake? Enter a working key to reveal it again.</p>
  </div>
</main>
<script src="/__qr.js"></script>
<script>
var TOKEN = {{TOKEN_JSON}};
var QR_DATA = {{QR_JSON}};
// EC level L: screens scan cleanly (no wear to guard against) and L packs
// payloads into fewer modules, so cameras resolve them from farther away.
function renderQR(id, data) {
  try {
    var qr = qrcode(0, 'L');
    qr.addData(data);
    qr.make();
    document.getElementById(id).innerHTML = qr.createSvgTag({cellSize: 4, scalable: true});
  } catch (e) {
    document.getElementById(id).textContent = 'QR error: ' + e;
  }
}
renderQR('qrcode-key', TOKEN);
renderQR('qrcode-link', QR_DATA);
function copyText(id, btn) {
  var text = document.getElementById(id).textContent.trim();
  var done = function () {
    var t = btn.textContent;
    btn.textContent = 'Copied';
    setTimeout(function () { btn.textContent = t; }, 1200);
  };
  var fallback = function () {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); done(); } catch (e) {}
    document.body.removeChild(ta);
  };
  if (navigator.clipboard) { navigator.clipboard.writeText(text).then(done, fallback); } else { fallback(); }
}
function sendTest() {
  var st = document.getElementById('test-status');
  st.textContent = 'Sending...';
  fetch('/{{TOPIC}}', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer ' + TOKEN,
      'Title': 'ntfy pup works',
      'Tags': 'white_check_mark'
    },
    body: 'If this shows up on your phone, your notification hub is live.'
  }).then(function (resp) {
    st.textContent = resp.ok
      ? 'Sent — check your phone. (If nothing arrives yet, finish steps 2 and 3 first; the message waits in the topic for 12h.)'
      : 'Server replied ' + resp.status + ' — is the key pasted correctly?';
  }).catch(function (e) {
    st.textContent = 'Could not reach the server: ' + e;
  });
}
function saveBridge() {
  var st = document.getElementById('bridge-status');
  st.textContent = 'Saving...';
  fetch('/bridge-token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token: document.getElementById('bridge-input').value.trim() })
  }).then(function (resp) {
    if (resp.ok) { st.textContent = 'Saved — the bridge picks it up within a minute.'; return; }
    resp.json().then(function (e) {
      st.textContent = e.error || 'Save failed (' + resp.status + ').';
    }, function () { st.textContent = 'Save failed (' + resp.status + ').'; });
  }).catch(function (e) { st.textContent = 'Error: ' + e; });
}
function clearBridge() {
  var st = document.getElementById('bridge-status');
  fetch('/bridge-token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token: '' })
  }).then(function (resp) {
    st.textContent = resp.ok ? 'Cleared.' : 'Clear failed (' + resp.status + ').';
    document.getElementById('bridge-input').value = '';
  }).catch(function (e) { st.textContent = 'Error: ' + e; });
}
</script>
</body>
</html>
`

const hiddenHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ntfy — key hidden</title>
<style>
  :root { --gold:{{GOLD}}; --bg:#171410; --card:#201c15; --line:#3a3325; --text:#efe9dc; --dim:#a89e88; }
  * { box-sizing:border-box; }
  body { margin:0; background:var(--bg); color:var(--text); font:16px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif; }
  main { max-width:520px; margin:0 auto; padding:28px 20px 72px; }
  h1 { font-size:1.4rem; }
  h1 .dot { color:var(--gold); }
  .card { background:var(--card); border:1px solid var(--line); border-radius:14px; padding:20px; margin-bottom:16px; }
  .muted { color:var(--dim); font-size:0.92rem; }
  .err { color:#e0704f; }
  input[type=password] { width:100%; background:#14110c; border:1px solid var(--line); color:var(--text); border-radius:8px; padding:10px 12px; font-size:0.95rem; margin:10px 0; }
  button { background:var(--gold); border:0; color:#171410; font-weight:600; border-radius:8px; padding:10px 16px; cursor:pointer; font-size:0.95rem; }
</style>
</head>
<body>
<main>
  <h1>ntfy on your Dogebox<span class="dot">.</span></h1>
  <div class="card">
    <p>The setup page's key is hidden. Enter a working device token to reveal it.</p>
    <p class="muted err">{{MESSAGE}}</p>
    <form method="post" action="/onboard/reveal">
      <input type="password" name="token" placeholder="Device token (tk_...)" autocomplete="off" autofocus>
      <button type="submit">Reveal</button>
    </form>
    <p class="muted">Lost every token? SSH into the box, remove <span class="mono">/opt/dogebox/pups/storage/&lt;pup&gt;/state/device-token</span> and restart the pup — a new one is generated on boot.</p>
  </div>
</main>
</body>
</html>
`
