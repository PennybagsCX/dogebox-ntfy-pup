package main

// Server-rendered onboarding pages. Placeholders are substituted with
// strings.NewReplacer in serveOnboarding/serveHidden. Kept as separate
// consts so frontdoor.go stays readable; no backticks anywhere (Go raw
// strings), no external assets except the embedded QR library at /__qr.js.

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
  main { max-width:680px; margin:0 auto; padding:28px 20px 72px; }
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
  #qrcode { background:#fff; display:inline-block; padding:12px; border-radius:12px; margin-top:8px; }
  #qrcode svg { display:block; width:200px; height:200px; }
  #qrcode::before { content:" "; }
  .muted { color:var(--dim); font-size:0.92rem; }
  a { color:var(--gold); }
  input[type=password],input[type=text] { flex:1; width:100%; background:#14110c; border:1px solid var(--line); color:var(--text); border-radius:8px; padding:10px 12px; font-size:0.95rem; margin-top:10px; }
  #test-status, #bridge-status { margin-top:10px; min-height:1.4em; color:var(--dim); }
  details.card { margin-top:0; }
  summary { cursor:pointer; color:var(--text); font-weight:600; }
  summary .muted { font-weight:400; }
  pre { background:#14110c; border:1px solid var(--line); border-radius:8px; padding:12px; overflow-x:auto; font-size:0.82rem; margin:10px 0; }
  ul { margin:8px 0 0; padding-left:20px; }
  li { margin:4px 0; }
  form.inline { display:inline; }
</style>
</head>
<body>
<main>
  <h1>ntfy on your Dogebox<span class="dot">.</span></h1>
  <p class="sub">Push notifications from anything on your box to your phone. Three steps, about two minutes.</p>

  <div class="card">
    <div class="step"><span class="num">1</span><h2>Install the ntfy app</h2></div>
    <ul>
      <li><a href="https://play.google.com/store/apps/details?id=io.heckel.ntfy" target="_blank" rel="noopener">Android — Google Play</a></li>
      <li><a href="https://apps.apple.com/app/ntfy/id1625396347" target="_blank" rel="noopener">iPhone — App Store</a></li>
    </ul>
  </div>

  <div class="card">
    <div class="step"><span class="num">2</span><h2>Add the health topic — scan the QR</h2></div>
    <p class="muted">Open the ntfy app → <b>Subscribe</b> → tap the QR icon and scan this. Your box's health alerts (pup crashes, recoveries, updates) arrive in this topic automatically.</p>
    <div id="qrcode"><noscript>Enable JavaScript to see the QR code, or copy the link below.</noscript></div>
    <div class="row"><code id="topic-link">{{QR_DATA}}</code><button class="ghost" onclick="copyText('topic-link', this)">Copy</button></div>
    <p class="muted">Topic: <span class="mono">{{TOPIC}}</span></p>
  </div>

  <div class="card">
    <div class="step"><span class="num">3</span><h2>Sign in with your device token</h2></div>
    <p class="muted">Your server is private — the app asks for credentials the first time it connects. Choose <b>Token</b> (or API key) login and paste this. One token works on all your phones.</p>
    <div class="row"><code id="token">{{TOKEN}}</code><button class="ghost" onclick="copyText('token', this)">Copy</button></div>
    <p class="muted">iPhone: in the ntfy app go to <i>Settings → Manage users → Add</i>, server <span class="mono">{{HOST_URL}}</span>, login type <b>API key / token</b>, then paste.</p>
  </div>

  <div class="card">
    <div class="step"><span class="num">4</span><h2>Test it</h2></div>
    <button onclick="sendTest()">Send a test notification</button>
    <div id="test-status"></div>
  </div>

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
    <p class="muted">Any ntfy integration works — scripts, cron, Home Assistant, Uptime Kuma, CI — just point it at this server with the token above. Extra topics are free-form: send to <span class="mono">/any-topic-name</span>, then subscribe to it in the app.</p>
  </details>

  <div class="card">
    <p class="muted">Credentials stay visible on this page until you hide them — anyone on your home network can read them, which is the trust model of a LAN-first box (never port-forward this; use Tailscale or similar for remote access). When your phones are paired:</p>
    <form method="post" action="/onboard/hide"><button class="ghost" type="submit">Hide credentials</button></form>
  </div>
</main>
<script src="/__qr.js"></script>
<script>
var TOKEN = {{TOKEN_JSON}};
var QR_DATA = {{QR_JSON}};
try {
  var qr = qrcode(0, 'M');
  qr.addData(QR_DATA);
  qr.make();
  document.getElementById('qrcode').innerHTML = qr.createSvgTag({cellSize: 4, scalable: true});
} catch (e) {
  document.getElementById('qrcode').textContent = 'QR error: ' + e;
}
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
      : 'Server replied ' + resp.status + ' — is the token pasted correctly?';
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
<title>ntfy — credentials hidden</title>
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
    <p>Credentials are hidden. Enter a working device token to reveal them.</p>
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
