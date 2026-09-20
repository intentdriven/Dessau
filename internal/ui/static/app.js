'use strict';

let state = null;

// ── helpers ──────────────────────────────────────────────
const $ = (id) => document.getElementById(id);

function bytes(n) {
  if (!n) return '—';
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${u[i]}`;
}

// foldRepoID is config.FoldRepoID: the one rule for when two repo ids name the
// same model. The panel joins pins against model ids, and a pin can be carried
// in a spelling the registry does not use — set before the model was
// downloaded — until the next save reconciles it, so the join folds like every
// other join on a repo id in this project does.
function foldRepoID(id) { return String(id ?? '').toLowerCase(); }

// pinLabel is the pill a model's card carries when it is pinned: which of the
// two things being pinned means right now, or '' when it is not. A model that
// is protected but that nothing has loaded says so, because an operator
// reading "pinned" would otherwise assume it was running.
function pinLabel(m, pinned, loaded) {
  if (m.state !== 'ready') return '';
  const want = foldRepoID(m.repo_id);
  if (!(pinned || []).some((id) => foldRepoID(id) === want)) return '';
  return loaded ? 'pinned' : 'pinned, not loaded';
}

// size is bytes() for a figure that is genuinely a measurement: nothing pinned
// is "0 B", not the em dash bytes() shows for a size it does not know. This
// line is read before anything is ticked, so zero is its opening state.
function size(n) { return n ? bytes(n) : '0 B'; }

// contextLabel renders a model's architectural maximum context for its card.
// The label says "max context" so the figure is not read as the window this
// Mac can hold at once, which is a smaller and separate number. The
// abbreviation rounds DOWN: rounding 262,143 up to 256K would show a token
// more than the model declares, and under-reporting is the safe direction for
// a reader sizing a prompt from the card. The result is composed from a
// number, so it is safe in the innerHTML the card is built from; anything
// that is not a positive number gives no label at all.
// tokensLabel writes a token count the way the cards do: whole kibitokens
// above a thousand, the figure itself below.
function tokensLabel(n) {
  return n >= 1024 ? `${Math.floor(n / 1024)}K` : String(n);
}

// contextLabel is the card's word on the model's windows: the declared one,
// and the served one when the operator has set it below. A model that
// declares none gets no label, never a zero. The served window is resolved
// the fold-aware way every other reader resolves it.
function contextLabel(m, config) {
  const n = m.context_length;
  if (typeof n !== 'number' || !Number.isFinite(n) || n <= 0) return '';
  const served = servedContext(config, m.repo_id, n);
  if (served > 0 && served < n) {
    return `context ${tokensLabel(n)} declared · ${tokensLabel(served)} served`;
  }
  return `max context ${tokensLabel(n)}`;
}

// modelInfoLine is the whole info line of a model's card: what it says
// about a download in flight, a failure, or a model on disk. It is a pure
// function of the model, so the panel's one piece of real logic can be
// tested without a DOM (see panel_test.go); renderModels does nothing with
// it but place the string it returns.
function modelInfoLine(m, config) {
  if (m.state === 'downloading') {
    const of = m.size_bytes ? ` of ${bytes(m.size_bytes)}` : '';
    return `downloading… ${m.progress.toFixed(0)}%${of}`;
  }
  if (m.state === 'failed') return escapeHtml(m.err || 'failed');
  const ctx = contextLabel(m, config);
  return ctx ? `${bytes(m.bytes)} · ${ctx}` : bytes(m.bytes);
}

// resourcesSummary adds up what the Models tab already knows, from the state
// snapshot alone — no walk, no second reader: how many models are downloaded
// and loaded, the disk they take against what the volume has left, and the
// memory budget against what is resident, naming the part still exiting. A
// pure function of the snapshot, so a test can hold every figure.
function resourcesSummary(state) {
  const st = state || {};
  const models = st.models || [];
  const resident = st.resident || [];
  const machine = st.machine || {};
  const ready = models.filter((m) => m.state === 'ready');
  const loaded = resident.filter((r) => r.state === 'loaded').length;
  const loading = resident.filter((r) => r.state === 'loading').length;
  const disk = ready.reduce((sum, m) => sum + (m.bytes || 0), 0);
  const out = {
    downloaded: ready.length, loaded, loading, disk,
    freeDisk: machine.free_disk || 0,
    budget: machine.budget || 0,
    resident: machine.resident_bytes || 0,
    exiting: machine.exiting_bytes || 0,
    stuck: machine.stuck_servers || 0,
  };
  const parts = [];
  parts.push(`${out.downloaded} model${out.downloaded === 1 ? '' : 's'} downloaded, ${out.loaded} loaded` +
    (out.loading ? `, ${out.loading} loading` : ''));
  if (out.downloaded > 0) {
    parts.push(`${bytes(out.disk)} on disk` + (out.freeDisk ? `, ${bytes(out.freeDisk)} free` : ''));
  }
  if (out.budget > 0) {
    let mem = `memory: ${bytes(out.resident)} of ${bytes(out.budget)} budget resident`;
    if (out.exiting > 0) mem += `, of which ${bytes(out.exiting)} still exiting`;
    if (out.stuck > 0) mem += ` (${out.stuck} server${out.stuck === 1 ? '' : 's'} stuck)`;
    parts.push(mem);
  }
  out.text = parts.join(' · ');
  return out;
}

async function api(path, opts) {
  const res = await fetch(path, opts);
  const text = await res.text();
  let body = null;
  try { body = text ? JSON.parse(text) : null; } catch { /* not json */ }
  if (!res.ok) {
    throw new Error(body?.error?.message || text || `HTTP ${res.status}`);
  }
  return body;
}

const postModel = (path, model) => api(path, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ model }),
});

// ── tabs ─────────────────────────────────────────────────
function showTab(name) {
  document.querySelectorAll('.tab').forEach(
    (t) => t.classList.toggle('active', t.dataset.tab === name));
  document.querySelectorAll('.panel').forEach(
    (p) => p.classList.toggle('active', p.id === `tab-${name}`));
  // The request rows are fetched rather than pushed: they are far too many to
  // ride the state snapshot the rest of the panel redraws from, so they are
  // asked for while the view that shows them is open and not otherwise.
  watchStats(name === 'stats');
}
document.querySelectorAll('.tab').forEach(
  (t) => t.addEventListener('click', () => showTab(t.dataset.tab)));

// ── live state ───────────────────────────────────────────
function connect() {
  const es = new EventSource('/api/events');

  es.onmessage = (e) => {
    state = JSON.parse(e.data);
    render();
    $('statusDot').className = 'dot up';
    $('statusText').textContent = `serving on :${state.config.port}`;
  };

  es.onerror = () => {
    $('statusDot').className = 'dot down';
    $('statusText').textContent = 'disconnected';
    // EventSource retries on its own; don't stack up extra connections.
  };
}

function render() {
  if (!state) return;
  // Shown wherever the panel is open, not only on the Statistics tab: on a
  // shared Mac the person whose requests are being recorded is not
  // necessarily the person who turned it on.
  $('recordingBadge').hidden = !(state.config && state.config.statistics);
  renderWarnings();
  renderSetup();
  renderModels();
  renderConnect();
  renderClients();
  renderPosture();
  renderSettings();
  // Independent of renderSettings, which returns early while the form is
  // being edited: the list of models to choose from is live state, not a
  // value the user is in the middle of typing.
  refreshOverrideModels();
  // And for the same reason: what the bridge is doing is a fact about the
  // server, not a field. Somebody who has just pasted a token is exactly the
  // person waiting to see it connect.
  renderBridgeState();
}

function renderWarnings() {
  const box = $('warnings');
  box.innerHTML = '';
  (state.warnings || []).forEach((w) => {
    const d = document.createElement('div');
    d.className = 'banner warn';
    d.innerHTML = `<div>⚠</div><div><strong>${escapeHtml(w)}</strong></div>`;
    box.appendChild(d);
  });
}

// setupHeading is what the setup banner says it is doing, and how far it has
// got. The proportion comes from the server's own SetupStatus — the same value
// `dessau install` renders in the terminal — so the two surfaces cannot drift.
// A status carrying no count at all still gets the heading it always had.
function setupHeading(s) {
  if (s.stage === 'failed') return 'Setup failed';
  const steps = Number(s.steps) || 0;
  if (steps > 0) return `Setting up: ${s.stage}… (${Number(s.step) || 0} of ${steps} done)`;
  return `Setting up: ${s.stage}…`;
}

function renderSetup() {
  const banner = $('setupBanner');
  const s = state.setup || {};
  const busy = !s.ready && s.stage !== 'idle' && s.stage !== 'ready';
  banner.hidden = s.ready || (!busy && s.stage !== 'failed');
  if (banner.hidden) return;

  // A failed setup must not read as a running one. The spinner and the "this
  // takes a few minutes" line describe work in progress; left in place beside
  // the words "Setup failed" they tell the operator to wait for something that
  // is not happening, and the panel offers no other signal that it has stopped.
  const failed = s.stage === 'failed';
  $('setupStage').textContent = setupHeading(s);
  $('setupSpinner').hidden = failed;
  $('setupFailMark').hidden = !failed;
  $('setupBlurb').textContent = failed
    ? 'Setup stopped and will not finish on its own. Quit Dessau and open it again to retry; if it keeps failing, the message below says why.'
    : 'Dessau is installing its own private Python and MLX. This happens once and takes a few minutes.';
  $('setupErr').textContent = s.err || '';
  banner.classList.toggle('failed', failed);
}

// ── my models ────────────────────────────────────────────
// graceWaitHint says when the two eviction-grace intervals contradict each
// other. A maximum wait below the grace does not shorten the wait, it disables
// the rule that stops one client starving another — a waiting request would be
// refused before its own wait could override a model's protection — so the
// server refuses the pair. Saying it beside the fields is better than answering
// the save with an error the operator has to decode.
//
// Blank means the default, which is what the server reads it as, so the
// comparison is on the resolved figures — and the defaults are the server's
// own, off the snapshot, rather than two Go constants copied into this file.
// A panel that has not been told them says nothing: a rule stated in figures
// the panel made up is worse than no hint at all. internal/ui/grace_test.go
// holds what this returns to config.validateGrace over the same pair.
function graceWaitHint(graceValue, maxWaitValue, defaults) {
  const d = defaults || {};
  const grace = parseInt(graceValue, 10) || d.eviction_grace_sec || 0;
  const maxWait = parseInt(maxWaitValue, 10) || d.eviction_max_wait_sec || 0;
  if (!grace || !maxWait) return '';
  if (maxWait >= grace) return '';
  return `A maximum wait of ${maxWait} s is shorter than the ${grace} s protection, `
    + 'which would refuse a waiting request before its own wait could override that '
    + 'protection. Raise the maximum to at least the protection.';
}

// intervalWords puts an interval into words, in minutes where the figure is a
// whole number of them and in seconds otherwise — rounding 90 seconds to
// "1.5 minutes" would state a figure the server does not hold. Empty for an
// interval the panel was not told, so whatever is written from it says nothing
// rather than naming a figure nobody sent.
function intervalWords(sec) {
  if (!sec) return '';
  if (sec % 60 !== 0) return `${sec} second${sec === 1 ? '' : 's'}`;
  const minutes = sec / 60;
  return `${minutes} minute${minutes === 1 ? '' : 's'}`;
}

// blankIsSentence puts a default interval into the words that go beside a
// field. Empty for a threshold the panel was not told, so the sentence
// disappears rather than naming a figure nobody sent.
function blankIsSentence(sec) {
  const words = intervalWords(sec);
  return words ? ` Blank is ${words}.` : '';
}

// idleThresholdWords is how long this Mac must have gone unasked before it
// counts as idle, in the words the status prose reads it in. It is the
// threshold in force — the setting where one is set, the server's default
// where it is not, which is config.EffectiveIdleThresholdSec on the Go side —
// and never a figure the panel holds a copy of (iss-2609190201317726). Told
// neither, it names the setting instead of a number: the sentences this goes
// into run on either side of it, so it cannot simply vanish the way the
// sentence beside the field does.
function idleThresholdWords(state) {
  const s = state || {};
  const c = s.config || {};
  const d = s.defaults || {};
  return intervalWords(c.idle_threshold_sec || d.idle_threshold_sec) || 'the idle threshold';
}

// renderIdleProse fills the figure in the Self-test hint, which is prose about
// the threshold in force rather than a hint about what a blank field means.
function renderIdleProse(state) {
  $('selfTestIdleFigure').textContent = idleThresholdWords(state);
}

// renderDefaults says, on the settings fields themselves, what leaving one
// blank resolves to. A blank field is the default, so the form has to name the
// figure that stands for — and it is the server's, off the snapshot, rather
// than Go constants written into the markup a second time
// (iss-2609190045306656 for the grace pair, iss-2609190146152463 for the idle
// threshold and the prose beside it). A panel that has not been told them
// offers no figure, the same silence graceWaitHint keeps: a placeholder the
// panel made up is worse than an empty one.
function renderDefaults(defaults) {
  const d = defaults || {};
  $('setGraceSec').placeholder = d.eviction_grace_sec ? String(d.eviction_grace_sec) : '';
  $('setGraceWait').placeholder = d.eviction_max_wait_sec ? String(d.eviction_max_wait_sec) : '';
  $('setIdleThreshold').placeholder = d.idle_threshold_sec ? String(d.idle_threshold_sec) : '';
  $('idleThresholdDefault').textContent = blankIsSentence(d.idle_threshold_sec);
}

function updateGraceHint() {
  const hint = graceWaitHint($('setGraceSec').value, $('setGraceWait').value,
    state && state.defaults);
  $('graceHint').textContent = hint;
  $('graceHint').hidden = hint === '';
}

// waitingLine says how many requests are queued for memory, and nothing at all
// when none are. A waiting request holds no model, so it appears on no card;
// without this the panel looks identical whether nothing is happening or four
// clients are queued behind a model that will not fall idle.
function waitingLine(waiting) {
  if (!waiting) return '';
  return waiting === 1
    ? '1 request is waiting for memory to free up.'
    : `${waiting} requests are waiting for memory to free up.`;
}

function renderModels() {
  $('resources').textContent = resourcesSummary(state).text;
  const list = $('modelList');
  // Live state, drawn here rather than in Settings, which stops redrawing
  // while the form is being edited.
  const queue = waitingLine(state.waiting || 0);
  $('graceQueue').textContent = queue;
  $('graceQueue').hidden = queue === '';
  const models = state.models || [];
  const resident = new Set((state.resident || []).map((r) => r.repo_id));
  // The set the pool is actually enforcing, not the stored settings: those two
  // are the same except for the moment between a model arriving and the pin
  // for it being reconciled, and this surface is the one an operator would act
  // on. A pinned model that nothing has loaded is marked too — "pinned" is a
  // fact about the model, not about its memory.
  const pinned = state.pinned || [];

  $('modelsEmpty').hidden = models.length > 0;
  list.innerHTML = '';

  models.forEach((m) => {
    const card = document.createElement('div');
    card.className = 'card';

    const loaded = resident.has(m.repo_id);
    let pill = '';
    if (m.state === 'ready')       pill = loaded
      ? '<span class="pill loaded">loaded</span>'
      : '<span class="pill ready">ready</span>';
    else if (m.state === 'failed') pill = '<span class="pill failed">failed</span>';
    const pinText = pinLabel(m, pinned, loaded);
    if (pinText) pill += `<span class="pill pinned">${pinText}</span>`;

    const info = modelInfoLine(m, state.config);
    const measured = measurementText(m, state.idle_jobs, state.probe_queue);

    card.innerHTML = `
      <div class="meta">
        <div class="name">${escapeHtml(m.repo_id)}${pill}</div>
        <div class="info">${info}</div>
        ${measured ? `<div class="info measured">${escapeHtml(measured)}</div>` : ''}
        ${m.state === 'downloading'
          ? `<div class="bar"><i style="width:${m.progress}%"></i></div>` : ''}
      </div>
      <div class="actions"></div>`;

    const actions = card.querySelector('.actions');

    if (m.state === 'downloading') {
      // Cancel abandons the download outright: stop it and remove the partial
      // files, rather than leaving a lingering "failed" row behind.
      actions.append(btn('Cancel', 'danger', () =>
        postModel('/api/models/delete', m.repo_id).catch(alertErr)));
    } else if (m.state === 'failed') {
      actions.append(btn('Retry', '', () =>
        postModel('/api/models/download', m.repo_id).catch(alertErr)));
      // A failed/partial download has nothing worth protecting — remove at once.
      actions.append(btn('Remove', 'danger', () =>
        postModel('/api/models/delete', m.repo_id).catch(alertErr)));
    } else {
      actions.append(loaded
        ? btn('Unload', 'ghost', () =>
            postModel('/api/models/unload', m.repo_id).catch(alertErr))
        : btn('Load', 'ghost', () =>
            postModel('/api/models/load', m.repo_id).catch(alertErr)));
      // The context probe: one run whatever the switch says, and adoption of
      // a current figure as the served window. A model that declares no
      // window has nothing to measure between.
      if (m.context_length > 0) {
        actions.append(btn('Measure now', 'ghost', () =>
          postModel('/api/models/measure', m.repo_id).catch(alertErr)));
      }
      if (m.measured && !m.measured.stale) {
        actions.append(btn('Use this window', 'ghost', () =>
          postModel('/api/models/adopt', m.repo_id).catch(alertErr)));
      }
      // A ready model is real data, so require a deliberate second click.
      actions.append(confirmBtn('Delete', 'Confirm?', 'danger', () =>
        postModel('/api/models/delete', m.repo_id).catch(alertErr)));
    }
    list.appendChild(card);
  });
}

// boundText says what stopped the probe's step above the measured window,
// in words: the model's own limit, or one of Dessau's bounds, in which case
// the figure is a floor.
function boundText(bound) {
  switch (bound) {
    case 'model': return 'the model refused above this: its own limit here';
    case 'prefill_deadline': return 'a floor: the prefill deadline stopped the probe first';
    case 'served_window': return 'a floor: the served window stopped the probe first';
    case 'memory_guard': return 'a floor: the memory guard stopped the probe first';
    default: return '';
  }
}

// staleText names why a measurement no longer holds.
function staleText(stale) {
  switch (stale) {
    case 'runtime': return 'the runtime changed';
    case 'budget': return 'the memory budget changed';
    case 'decode_concurrency': return 'the decode concurrency changed';
    case 'served_context': return 'the served window changed';
    default: return 'settings changed';
  }
}

// measurementText is the card's line about the context probe: the run in
// progress or what held it back, else the measurement and its bound, else
// that a probe was interrupted, else nothing. A pure function a test holds.
function measurementText(m, jobs, queue) {
  const j = jobs || {};
  const same = (a, b) => (a || '').toLowerCase() === (b || '').toLowerCase();
  if (j.job === 'context-probe' && same(j.model, m.repo_id)) {
    return `Measuring: ${j.step || 'starting'}`;
  }
  const queued = (queue || []).some((q) => same(q, m.repo_id)) || same(j.due, m.repo_id);
  if (queued) {
    const held = { in_flight: 'a request is in flight', waiting: 'a caller is waiting for a model',
      downloading: 'a download is running', recent: 'a request was served recently',
      no_room: 'the model does not fit the memory budget' }[j.held_by];
    return held ? `Measurement waiting: ${held}` : 'Measurement queued for the next idle minute';
  }
  const mm = m.measured;
  if (mm) {
    const tokens = `${mm.window.toLocaleString()} tokens`;
    if (mm.stale) return `Measured ${tokens}, now stale: ${staleText(mm.stale)}; measure again`;
    return `Measured ${tokens} — ${boundText(mm.bound)}`;
  }
  if (m.probe_incomplete) return 'Measurement incomplete: the last probe was interrupted; press Measure now to run it again';
  return '';
}

function btn(label, cls, onClick) {
  const b = document.createElement('button');
  b.textContent = label;
  if (cls) b.className = cls;
  b.addEventListener('click', onClick);
  return b;
}

// confirmBtn is an inline two-click confirmation. The first click arms the
// button (it shows confirmLabel for 3s); a second click within that window runs
// the action. This replaces window.confirm, which browsers let the user
// permanently suppress — silently turning destructive buttons into no-ops.
function confirmBtn(label, confirmLabel, cls, onConfirm) {
  const b = document.createElement('button');
  b.textContent = label;
  if (cls) b.className = cls;
  let armed = false;
  let timer = null;
  b.addEventListener('click', () => {
    if (armed) {
      clearTimeout(timer);
      armed = false;
      b.textContent = label;
      b.classList.remove('armed');
      onConfirm();
      return;
    }
    armed = true;
    b.textContent = confirmLabel;
    b.classList.add('armed');
    timer = setTimeout(() => {
      armed = false;
      b.textContent = label;
      b.classList.remove('armed');
    }, 3000);
  });
  return b;
}

// ── search ───────────────────────────────────────────────
$('searchForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const q = $('searchInput').value.trim();
  const box = $('searchResults');
  box.innerHTML = '<p class="hint">Searching…</p>';
  try {
    const data = await api(`/api/search?q=${encodeURIComponent(q)}`);
    renderSearch(data);
  } catch (err) {
    box.innerHTML = `<p class="msg err">${escapeHtml(err.message)}</p>`;
  }
});

// pipelineLabel is what a search result says it is: HuggingFace's own pipeline
// tag, or "no tag" when the Hub has none for that repo. "no tag" is a fact
// about the repository — plenty of good models carry none — so it is said
// rather than left as a blank the reader has to interpret.
function pipelineLabel(m) {
  const tag = (m && m.pipeline_tag) || '';
  return tag ? escapeHtml(tag) : 'no tag';
}

function renderSearch(data) {
  const box = $('searchResults');
  const results = (data && data.results) || [];
  const machine = data && data.machine;
  const hidden = (data && data.hidden) || 0;
  box.innerHTML = '';

  // A note describing this Mac and how many models were hidden as too large.
  if (machine && machine.total_ram) {
    const note = document.createElement('p');
    note.className = 'hint';
    let text = `This Mac: ${bytes(machine.total_ram)} RAM · ${bytes(machine.free_disk)} free.`;
    if (hidden > 0) {
      text += ` ${hidden} model${hidden === 1 ? '' : 's'} hidden (too large to run or store here).`;
    } else {
      text += ' Showing models that fit.';
    }
    note.textContent = text;
    box.appendChild(note);
  }

  if (!results.length) {
    const p = document.createElement('p');
    p.className = 'hint';
    p.textContent = hidden > 0
      ? 'No models small enough for this Mac matched your search.'
      : 'No models found.';
    box.appendChild(p);
    return;
  }
  results.forEach((m) => {
    const card = document.createElement('div');
    card.className = 'card';
    const quant = m.quantization
      ? `<span class="pill quant">${escapeHtml(m.quantization)}</span>` : '';
    const size = m.size_bytes ? ` · ${bytes(m.size_bytes)}` : '';
    const kind = pipelineLabel(m);
    card.innerHTML = `
      <div class="meta">
        <div class="name">${escapeHtml(m.id)}${quant}</div>
        <div class="info">${kind} · ${(m.downloads || 0).toLocaleString()} downloads · ${m.likes || 0} likes${size}</div>
      </div>
      <div class="actions"></div>`;

    const actions = card.querySelector('.actions');
    if (m.local_state === 'ready') {
      const b = btn('Downloaded', 'ghost', () => {});
      b.disabled = true;
      actions.append(b);
    } else if (m.local_state === 'downloading') {
      const b = btn('Downloading…', 'ghost', () => {});
      b.disabled = true;
      actions.append(b);
    } else {
      actions.append(btn('Download', '', async (ev) => {
        ev.target.disabled = true;
        ev.target.textContent = 'Starting…';
        try {
          await postModel('/api/models/download', m.id);
          showTab('models');
        } catch (err) {
          alertErr(err);
          ev.target.disabled = false;
          ev.target.textContent = 'Download';
        }
      }));
    }
    box.appendChild(card);
  });
}

// ── connect ──────────────────────────────────────────────
// endpointOf reads one entry of state.endpoints: the URL a client points at,
// and the kind of network that address sits on. The second is blank on every
// Mac with no private network, which is most of them.
function endpointOf(e) {
  return { url: (e && e.url) || '', network: (e && e.network) || '' };
}

// endpointLine is one row: the URL, and beside it the network Dessau found
// the address on. The mark is what the server observed and not what it
// guarantees — it says which network, never what that network is worth
// (adr-2609081118587999) — and it sits in its own element, so nothing that
// hands the operator something to paste can pick it up.
function endpointLine(ep) {
  const mark = ep.network ? `<span class="pill">${escapeHtml(ep.network)}</span>` : '';
  return `<span>${escapeHtml(ep.url)}</span>${mark}`;
}

// ── clients ──────────────────────────────────────────────
// Every chat client paired with this server (adr-2609182357322050). The name is
// chosen by whoever paired, and under first-come pairing that is anything on
// the network — so every field of a row is set with textContent and none of it
// is ever interpolated into markup. The panel is loopback-only and asks for no
// credential, so a name that ran as script would be running same-origin on the
// whole settings API.

// clientPaired is when a client paired, in the reader's own locale.
function clientPaired(at) {
  return at ? new Date(at * 1000).toLocaleString() : 'unknown';
}

// clientSeen is when this server last heard from a client.
//
// The sighting is held in memory rather than in the settings file — writing it
// would fsync config.json once per request — so a server that has just started
// has heard from nobody, and says that rather than showing a time nobody
// recorded.
function clientSeen(at) {
  return at ? new Date(at * 1000).toLocaleString() : 'not since this server started';
}

function renderClients() {
  const fp = $('serverFingerprint');
  fp.textContent = state.server_fingerprint
    || 'This server has no certificate of its own, so no client can pair.';

  const box = $('clients');
  box.innerHTML = '';
  const clients = state.clients || [];
  if (!clients.length) {
    const p = document.createElement('p');
    p.className = 'hint';
    p.textContent = 'No chat client has paired with this server.';
    box.appendChild(p);
    return;
  }
  clients.forEach((c) => {
    const row = document.createElement('div');
    row.className = 'endpoint';
    const name = document.createElement('div');
    const strong = document.createElement('strong');
    strong.textContent = c.name;
    const detail = document.createElement('div');
    detail.className = 'info';
    // Two clients may choose one name — that is precisely what an impostor
    // does — so the fingerprint is shown in full beside it, and it is the
    // fingerprint that Revoke acts on.
    detail.textContent = `${c.spki} · paired ${clientPaired(c.paired_at)}`
      + ` · last seen ${clientSeen(c.last_seen)}`;
    name.appendChild(strong);
    name.appendChild(detail);
    row.appendChild(name);
    row.append(confirmBtn('Revoke', 'Revoke?', 'ghost', () => {
      api('/api/clients/revoke', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fingerprint: c.spki }),
      }).catch(alertErr);
    }));
    box.appendChild(row);
  });
}

function renderConnect() {
  const box = $('endpoints');
  box.innerHTML = '';
  const eps = (state.endpoints || []).map(endpointOf);
  eps.forEach((ep) => {
    const row = document.createElement('div');
    row.className = 'endpoint';
    row.innerHTML = endpointLine(ep);
    row.append(btn('Copy', 'ghost', () => navigator.clipboard.writeText(ep.url)));
    box.appendChild(row);
  });

  // The examples take the URL field, never the rendered row: a base URL with
  // a network mark appended to it is not a base URL.
  const base = (eps.length ? eps[0].url : '') || `http://localhost:${state.config.port}/v1`;
  const model = (state.models.find((m) => m.state === 'ready') || {}).repo_id
    || 'mlx-community/Qwen3-8B-4bit';
  const authCurl = state.config.api_key
    ? ` \\\n  -H "Authorization: Bearer YOUR_KEY"` : '';

  $('curlExample').textContent =
`curl ${base}/chat/completions \\
  -H "Content-Type: application/json"${authCurl} \\
  -d '{
    "model": "${model}",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`;

  const authPy = state.config.api_key ? '"YOUR_KEY"' : '"not-needed"';
  $('pyExample').textContent =
`from openai import OpenAI

client = OpenAI(base_url="${base}", api_key=${authPy})

resp = client.chat.completions.create(
    model="${model}",
    messages=[{"role": "user", "content": "Hello!"}],
)
print(resp.choices[0].message.content)`;
}

// ── posture ──────────────────────────────────────────────
// One page that says what is on (itd-2609081718534201). Every line below is
// derived from the state snapshot and from no other source: no fetch, no
// second reading, no new way to be wrong. Each line names the snapshot fields
// it read in `reads`, which internal/ui's tests hold to the Go type the
// control plane publishes; a line with no fields is a fact about the binary
// rather than an observation of this server, and there are two of those.
//
// What the RUNNING bind is — whether it reaches another machine, which mode
// is in force, whether it is the wildcard — is read from state.bind, which
// the control plane fills from the plan the sockets were acquired under. It
// is never inferred from the stored configuration, which a save changes
// before a restart applies it, nor from the endpoint list, which omits
// addresses it cannot name while the sockets answer on them.
//
// The lines say what is, in the present tense, and say where Dessau's view
// stops. They state which network an address is on and nothing about what
// that network is worth (adr-2609081118587999 rule 1), and nothing here is an
// input to any decision the server makes: the page reports and gates nothing.

// advertising reads the decision the process made at start about the
// Bonjour advert, which the bind state carries: the advert is started once
// and stopped at shutdown, so the stored setting — which a save changes at
// once — is not what is running. It returns which of the three reasons in
// that rule (the setting, the mode, the reach) keeps the advert off, or ''
// when it is on.
function advertising(state) {
  const bind = state.bind || {};
  if (bind.advertising) return '';
  if (bind.mode_in_force === 'private-network') return 'mode';
  if (!bind.reaches_other_machines) return 'bind';
  return 'setting';
}

// postureLines is the page: an array of {id, heading, text, reads}.
function postureLines(state) {
  const c = state.config || {};
  const bind = state.bind || {};
  const eps = (state.endpoints || []).map(endpointOf).filter((ep) => ep.url);
  const urls = eps.map((ep) => ep.url);
  const lines = [];
  const list = (items) => items.length < 2 ? items.join('')
    : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`;
  const reaches = !!bind.reaches_other_machines;

  // Who can reach it: what the running bind acquired, and the addresses the
  // list can name. The bind is observable; reachability is not, and the
  // line says so. Under the wildcard the list is not exhaustive — it names
  // what it can — and its first entry is a name and not an address.
  let reach;
  if (!reaches) {
    reach = `This server answers on this Mac and on no other address: ${list(urls)}. ` +
      'A request from another machine reaches nothing.';
  } else if (bind.wildcard) {
    reach = `This server answers on every address this Mac holds. The ones Dessau can name are ${list(urls)}; ` +
      "a name ending in .local is this Mac's name on the local network and not an address. ";
  } else if (bind.bound) {
    reach = `This server answers on ${bind.bound} and on this Mac; the addresses clients can use are ${list(urls)}. `;
  } else {
    // A bound address the panel cannot write as a URL — one carrying an
    // interface zone — is answered on all the same, and the list below
    // leaves it out; the page says so rather than leaving a gap.
    reach = 'This server answers on this Mac and on one more address, which Dessau cannot write as a URL; ' +
      `the addresses it can name are ${list(urls)}. `;
  }
  if (reaches) {
    reach += 'Which machines can reach an address is decided by the network it is on, and Dessau does not see that.';
  }
  if (bind.refusal) reach += ` The bind narrowed to this Mac: ${bind.refusal}.`;
  lines.push({ id: 'reach', heading: 'Who can reach it', text: reach,
    reads: ['endpoints.url', 'bind.reaches_other_machines', 'bind.wildcard', 'bind.bound', 'bind.refusal'] });

  // The private network, when there is one: the mark says which network, and
  // this line says what that mark cannot see. Sharing and public tunnelling
  // both change who reaches the address and neither touches the interface.
  const marked = eps.filter((ep) => ep.network);
  const chosen = bind.mode === 'private-network';
  const running = bind.mode_in_force === 'private-network';
  if (marked.length || chosen || running) {
    let text = marked.length
      ? `${list(marked.map((ep) => ep.url))} ${marked.length === 1 ? 'is' : 'are'} on a private network. `
      : 'No address on a private network is being answered on. ';
    text += 'Dessau reads that from the interface an address sits on and the range it falls in, and reads ' +
      'nothing from the network itself. Whether that network has since been shared with machines you do not ' +
      'own, or whether a feature of the network publishes this port to the internet, Dessau cannot see, ' +
      'and neither changes the address or the mark.';
    if (running) {
      text += bind.selected
        ? ` The private-network choice selected ${bind.selected}.`
        : ' The private-network choice is in force and selected no address, so the server answers on this Mac.';
    } else if (chosen) {
      text += ' The private-network choice is saved and is not in force until Dessau next starts.';
    }
    lines.push({ id: 'private', heading: 'The private network', text,
      reads: ['endpoints.url', 'endpoints.network', 'bind.mode', 'bind.mode_in_force', 'bind.selected'] });
  }

  lines.push({ id: 'transport', heading: 'What carries a request', reads: [],
    text: 'Every address is plain HTTP. Dessau does no TLS: whatever protection a request has on its way ' +
      'here comes from the network it travelled, and Dessau does not see that either.' });

  lines.push({ id: 'panel', heading: 'This control panel', reads: [],
    text: 'This panel, and the API it is drawn from, answer on this Mac alone whichever bind is chosen. ' +
      'Every account on this Mac can open it.' });

  // Two key lines, never one: withAuth admits a loopback connection with a
  // loopback Host without the key, so this Mac — another account on it
  // included — is served without it while every other machine is refused.
  const keySet = !!c.api_key;
  let fromNetwork;
  if (keySet) {
    fromNetwork = reaches
      ? 'A request arriving from another machine has to carry the API key. A key is set.'
      : 'A key is set. The bind reaches no other machine, so nothing arrives from one to carry it.';
  } else {
    fromNetwork = reaches
      ? 'No API key is set. A request arriving from another machine is served without one.'
      : 'No API key is set, and the bind reaches no other machine.';
  }
  lines.push({ id: 'key-network', heading: 'A request from another machine', text: fromNetwork,
    reads: ['config.api_key', 'bind.reaches_other_machines'] });
  lines.push({ id: 'key-local', heading: 'A request from this Mac', reads: ['config.api_key'],
    text: keySet
      ? 'A request from this Mac to a loopback address is served without the key, and that includes a ' +
        'request from another account on this Mac. The key applies to the network and not to this Mac.'
      : 'A request from this Mac is served without a key, as every request is.' });

  // The announcement: the one thing Dessau sends to every machine on the
  // local network, and what it carries. The service is named after this Mac
  // and published under a name of Dessau's own, never the Mac's own .local
  // name (internal/discovery); with no name to read, the advert says dessau.
  const off = advertising(state);
  const name = state.hostname || 'dessau';
  let announce;
  if (!off) {
    announce = 'Dessau is announcing this server to every machine on the local network, as a Bonjour ' +
      `service named after this Mac's name, ${name}, shortened where it is too long for a service name. ` +
      `The announcement carries this Mac's addresses, port ${bind.port}, how many models are ready, whether ` +
      'a key is required, and the fixed words saying it speaks the OpenAI API under /v1. It carries no ' +
      'model names and no key.';
  } else {
    const why = {
      setting: 'announcing was switched off when Dessau started',
      mode: 'the announcement travels over the local network, which the private-network choice excludes',
      bind: 'the bind reaches no other machine',
    }[off];
    announce = `Dessau is not announcing this server: ${why}.`;
  }
  announce += ' This line reads the decision made when Dessau started, from the setting and the bind then in ' +
    'force; a setting changed since then takes effect at the next start, and an announcement that failed to ' +
    'start is reported in the log and not here.';
  lines.push({ id: 'announce', heading: 'The local network', text: announce,
    reads: ['bind.advertising', 'bind.port', 'bind.mode_in_force', 'bind.reaches_other_machines', 'hostname'] });

  // The level is applied live — a save moves it on the next line — so the
  // stored setting is the level in force, unlike the bind and the advert.
  const level = c.log_level || 'sparse';
  lines.push({ id: 'log', heading: 'The request log', reads: ['config.log_level'],
    text: "Each request to the API's endpoints is written to the server log as its method, path, status and " +
      'duration. The line carries no client address, no prompt, no answer and no key. The log is at the ' +
      `${level} level` + (level === 'detailed'
        ? ', which adds to each line the figures the sparse level leaves out'
        : ', one line for each thing that mattered') +
      ", and is kept in the logs folder of this account's Dessau data folder, in a file created for this " +
      'account alone.' });

  // What is recorded, where, and for how long. The store's figures ride the
  // snapshot while recording is on, so their absence is the observation — and
  // a store that could not be opened says so before any figure, because a
  // refused store reported as an empty one would have the operator believing
  // records were accumulating.
  let stats;
  if (c.statistics) {
    const store = state.stats_store || {};
    let kept;
    if (store.refused) {
      kept = 'Dessau could not open the store where records are kept, so nothing is on disk and the ' +
        'figures in the Statistics tab are held in memory; its own log says why';
    } else if (store.oldest) {
      kept = `records from ${new Date(store.oldest * 1000).toISOString().slice(0, 10)} onwards, ` +
        `${bytes(store.bytes)} in ${store.files} file${store.files === 1 ? '' : 's'}`;
    } else {
      kept = 'none kept yet';
    }
    if (!store.refused && store.stalled) {
      kept += ', and the disk did not answer in time, so the newest records may be missing from that';
    }
    stats = 'Request statistics are being recorded on this Mac: for each request, the model, when it ' +
      'arrived, how it ended, whether it streamed, the tokens in and out, and how long it took; and beside ' +
      'those, when a model was loaded or evicted, and the settings in force. A record holds no prompt, no ' +
      `answer, no key and no client address. Records are kept for ${c.stats_months} months and within ` +
      `${bytes(c.stats_max_bytes)}, in this account's Dessau data folder; ${kept}. Anyone who can open ` +
      'this panel can read them, which is every account on this Mac.';
  } else {
    stats = 'Request statistics are off: no request is recorded.';
  }
  lines.push({ id: 'stats', heading: 'Request statistics', text: stats,
    reads: ['config.statistics', 'config.stats_months', 'config.stats_max_bytes',
      'stats_store.refused', 'stats_store.oldest', 'stats_store.bytes', 'stats_store.files', 'stats_store.stalled'] });

  // The self-test loads models on its own while the Mac is idle, which is a
  // thing that can be on; the page says so from the setting, which applies
  // the moment it is saved. The interval it names is the threshold in force,
  // off the snapshot: the setting where one is set, the served default where
  // it is not, and never a figure written into this page (iss-2609190201317726).
  const idleFor = intervalWords(c.idle_threshold_sec || (state.defaults || {}).idle_threshold_sec)
    || 'the idle threshold';
  const selfTest = c.self_test
    ? `The self-test is on: while nothing has asked this Mac for a model for ${idleFor} and ` +
      'nothing is downloading, Dessau loads one of its models at a time where it fits beside ' +
      'what is loaded, measures it with a fixed set of prompts, and unloads what it loaded. A request ' +
      'from anyone ends the run. The figures go to a file in this account\'s Dessau data folder and ' +
      'hold no prompt and no answer.'
    : 'The self-test is off: Dessau loads no model on its own.';
  lines.push({ id: 'selftest', heading: 'Self-test', text: selfTest,
    reads: ['config.self_test', 'config.idle_threshold_sec', 'defaults.idle_threshold_sec'] });

  return lines;
}

// postureShown is the markup last drawn. The snapshot arrives every couple of
// seconds, and prose someone is reading or selecting must not be torn down
// and rebuilt under them when nothing in it changed.
let postureShown = '';

// renderPosture draws the lines into the view's own container and nowhere
// else: the page is somewhere the operator goes, and it interrupts nothing.
function renderPosture() {
  const html = postureLines(state).map((l) =>
    `<div class="fact"><h3>${escapeHtml(l.heading)}</h3><p>${escapeHtml(l.text)}</p></div>`).join('');
  if (html === postureShown) return;
  postureShown = html;
  $('posture').innerHTML = html;
}

// ── settings ─────────────────────────────────────────────
let settingsTouched = false;
document.querySelectorAll('#settingsForm input, #settingsForm select')
  .forEach((el) => el.addEventListener('input', () => { settingsTouched = true; }));

// The sampling fields, paired with the config keys they carry. Each is a
// number or nothing at all: blank means "pass no flag", which is not the same
// as zero — zero is a real temperature.
const SAMPLING_FIELDS = [
  { key: 'temperature', input: 'setTemp',      over: 'ovTemp',      integer: false },
  { key: 'top_p',       input: 'setTopP',      over: 'ovTopP',      integer: false },
  { key: 'top_k',       input: 'setTopK',      over: 'ovTopK',      integer: true },
  { key: 'min_p',       input: 'setMinP',      over: 'ovMinP',      integer: false },
  { key: 'max_tokens',  input: 'setMaxTokens', over: 'ovMaxTokens', integer: true },
];

// The per-model overrides being edited. Held here rather than read back off
// the form, so removing one and saving is a single action.
let overrides = {};

// numberOrNull reads one numeric input. A blank field is null — the key is
// still sent, so clearing a field clears the stored value.
function numberOrNull(id, integer) {
  const raw = $(id).value.trim();
  if (raw === '') return null;
  const n = Number(raw);
  if (!Number.isFinite(n)) return null;
  return integer ? Math.trunc(n) : n;
}

function readSampling(which) {
  const out = {};
  SAMPLING_FIELDS.forEach((f) => { out[f.key] = numberOrNull(f[which], f.integer); });
  return out;
}

function writeSampling(which, values) {
  const v = values || {};
  SAMPLING_FIELDS.forEach((f) => {
    $(f[which]).value = (v[f.key] === undefined || v[f.key] === null) ? '' : v[f.key];
  });
}

// isEmptySampling reports whether nothing at all is set, so "set override"
// with every field blank removes the override instead of storing an empty one.
function isEmptySampling(values) {
  return SAMPLING_FIELDS.every((f) => values[f.key] === null || values[f.key] === undefined);
}

function describeSampling(values) {
  return SAMPLING_FIELDS
    .filter((f) => values[f.key] !== null && values[f.key] !== undefined)
    .map((f) => `${f.key} ${values[f.key]}`)
    .join(' · ');
}

// extraBindOption reports the bind address the select has to be given an
// option for before the stored host can be assigned to it, or null when it
// already offers that address.
//
// HTMLSelectElement.value has no notion of a value the element does not carry:
// assigning one sets selectedIndex to -1 and leaves .value the empty string. A
// host config.json accepts and this select never offered — "localhost", "[::1]",
// one specific address — was therefore posted back as "" on every save, a save
// that changed nothing about the bind included, and refused with "host must not
// be empty": a field the pane never showed the operator as wrong, and nothing
// on the pane saveable again until they edited config.json by hand.
function extraBindOption(host, offered) {
  if (!host || offered.includes(host)) return null;
  return host;
}

// PRIVATE_BIND is the third choice in the bind select. It is a MODE, not an
// address: it is posted in bind_mode and never in host, because a word in host
// passes the server's host validation as a name and then fails to listen.
const PRIVATE_BIND = 'private-network';

// bindSelectValue is what the select shows for a configuration: the mode when
// a mode is in force, and the bind address otherwise.
function bindSelectValue(c) {
  return c.bind_mode === PRIVATE_BIND ? PRIVATE_BIND : c.host;
}

// bindSelectBody is the pair of fields a save posts for the chosen bind.
//
// Choosing the mode leaves host as it was stored, so that switching the mode
// off puts back the bind the operator had — and so that a save is never
// refused, or silently changed, over a field they did not touch.
function bindSelectBody(chosen, storedHost) {
  return chosen === PRIVATE_BIND
    ? { host: storedHost, bind_mode: PRIVATE_BIND }
    : { host: chosen, bind_mode: '' };
}

// privateBindLabel says what the mode would bind, or why it cannot.
//
// It states which network an address is on and nothing about what that network
// is worth: Dessau cannot see whether the network has been published to the
// internet or shared with machines the operator does not own, and neither
// transition touches the address.
function privateBindLabel(bind) {
  const found = (bind && bind.candidates) || [];
  // What the running mode bound comes first: the address a private network
  // hands out can change under a running server, and the pane has to name the
  // one being answered on rather than the one that matches now.
  const bound = (bind && bind.selected) || '';
  if (bound) return `A private network (${bound}) — and this Mac`;
  if (found.length === 1) return `A private network (${found[0]}) — and this Mac`;
  // Named rather than counted: a refusal the operator can act on is one that
  // says which addresses it would not choose between.
  if (found.length > 1) return `A private network — ${found.join(', ')} all match, so Dessau will not choose`;
  return 'A private network — no matching address on this Mac';
}

// privateBindDisabled reports whether the choice cannot be made at all.
//
// A mode already in force stays selectable with nothing to select: a select
// cannot show a value it does not offer, so disabling it there would blank the
// control and post the mode away on the next save — the fault the bind select
// was fixed for.
function privateBindDisabled(bind) {
  const found = (bind && bind.candidates) || [];
  return found.length === 0 && (!bind || bind.mode !== PRIVATE_BIND);
}

// bindNoticeText is what the pane says when the bind narrowed: the reason the
// server gave, or nothing at all when nothing was refused.
function bindNoticeText(bind) {
  return (bind && bind.refusal) || '';
}

// renderBindMode labels the third choice with what it would bind, and takes it
// away when there is nothing to bind.
function renderBindMode(select, bind) {
  for (const opt of Array.from(select.options)) {
    if (opt.value !== PRIVATE_BIND) continue;
    opt.textContent = privateBindLabel(bind);
    opt.disabled = privateBindDisabled(bind);
  }
}

// renderBindOptions makes the bind-address select offer the host in force,
// labeled with the address itself, so the value round-trips and the pane shows
// the bind Dessau is actually serving on rather than a blank control.
//
// The option added last time is dropped first. renderSettings runs on every
// live update, so without that a pane left open across a bind change would
// collect one option per address it has held, and offer the operator addresses
// this Mac no longer serves on.
function renderBindOptions(select, host) {
  for (const opt of Array.from(select.options)) {
    if (opt.dataset.extraBind) opt.remove();
  }
  const extra = extraBindOption(host, Array.from(select.options, (o) => o.value));
  if (extra === null) return;
  const opt = document.createElement('option');
  opt.value = extra;
  opt.textContent = extra;
  opt.dataset.extraBind = 'true';
  select.appendChild(opt);
}

// renderBridgeState says what the Discord bridge is doing, under the switch.
//
// Four states and nothing else: off, connecting, connected, or stopped with
// the reason Discord gave, each with the moment the bridge last connected
// where there is one. The reason is the server's own sentence about a
// credential or a configuration and never carries the token, which the panel
// is never sent in the first place.
function renderBridgeState() {
  const el = $('discordState');
  if (!el) return;
  el.textContent = bridgeStateText(state.bridge || {});
}

// bridgeStateText is the sentence the card shows, computed from the snapshot's
// bridge alone so it can be asserted without a DOM.
//
// The moment the bridge last connected is shown in every state that has one,
// and not only while it is connected: a session that has dropped and is being
// re-opened is exactly when somebody wants to know when the bot last worked.
// The moment is absent until the bridge has connected once under the current
// switch, and the sentence then says only what it is doing.
function bridgeStateText(b) {
  const at = b.since ? new Date(b.since * 1000).toLocaleString() : '';
  switch (b.state) {
    case 'connected':
      return at ? `Connected since ${at}.` : 'Connected.';
    case 'connecting':
      return at ? `Last connected at ${at}; reconnecting\u2026` : 'Connecting to Discord\u2026';
    case 'stopped': {
      const why = b.reason ? `Stopped: ${b.reason}.` : 'Stopped.';
      return at ? `${why} Last connected at ${at}.` : why;
    }
    default:
      return 'The bridge is off.';
  }
}

function renderSettings() {
  // Don't stomp on what the user is typing while live updates arrive.
  if (settingsTouched) return;
  const c = state.config;
  // Before the assignment, never after: an unoffered value assigns as "".
  renderBindOptions($('setHost'), c.host);
  renderBindMode($('setHost'), state.bind);
  const notice = bindNoticeText(state.bind);
  $('bindNotice').textContent = notice;
  $('bindNotice').hidden = notice === '';
  $('setHost').value = bindSelectValue(c);
  $('setPort').value = c.port;
  $('setTLSPort').value = c.tls_port || 0;
  $('setKey').value  = c.api_key || '';
  // The stored value, not what is being advertised right now: this is the
  // control, and the decision the running server took at start is on the
  // posture page. The two differ exactly while a change is waiting for a
  // restart, and the hint beside the box is what says so.
  $('setAdvertise').checked = !!c.advertise;
  $('setIdle').value = c.idle_timeout_sec;
  $('setConc').value = c.decode_concurrency;
  // The stored value in the box, and the notice beneath it for the window in
  // which that is not what the pool is running with.
  updateConcurrencyNotice();
  // From the machine object, not from the stored setting: what is enforced is
  // what the pool holds, and the stored setting is blank while the default is
  // in force.
  $('setBudget').value = budgetFieldValue(state.machine);
  updateBudgetHint();
  $('setHF').value   = c.hf_token || '';
  $('setDiscordBridge').checked = !!c.discord_bridge;
  $('setDiscordToken').value = c.discord_token || '';
  $('setGrace').checked = !!c.eviction_grace;
  // Blank rather than zero for an unset interval: blank is how this form says
  // "the default", and the placeholder gives the figure that stands for.
  renderDefaults(state.defaults);
  renderIdleProse(state);
  $('setGraceSec').value = c.eviction_grace_sec || '';
  $('setGraceWait').value = c.eviction_max_wait_sec || '';
  updateGraceHint();
  // The rule in force, which is what the server answers with — never blank
  // while a default stands behind it, because this form posts back what it
  // shows and two blank fields mean "test nothing".
  const rule = c.chat_rule || {};
  $('setChatPipelines').value = (rule.pipeline_tags || []).join(', ');
  $('setChatTags').value = (rule.required_tags || []).join(', ');
  $('setContextProbe').checked = !!c.context_probe;
  $('setIdleThreshold').value = c.idle_threshold_sec || '';
  $('setSelfTest').checked = !!c.self_test;
  $('setStats').checked = !!c.statistics;
  $('setStatsMonths').value = c.stats_months;
  // Typed in megabytes and stored in bytes, which is how every other size in
  // this panel is shown.
  $('setStatsMB').value = Math.round((c.stats_max_bytes || 0) / (1024 * 1024));
  renderStatsStore();
  // An absent value is the default, not a blank: a settings file written
  // before this field existed, and a fresh install, both mean sparse.
  $('setLogLevel').value = c.log_level || 'sparse';
  writeSampling('input', c.sampling);
  overrides = {};
  Object.entries(c.models || {}).forEach(([id, ms]) => {
    if (ms && ms.sampling) overrides[id] = ms.sampling;
  });
  renderOverrides();
  renderMergeSwitches();
  renderPinSwitches();
  renderContextFields();
}

// renderPinSwitches draws one box per downloaded model, and the figure that
// says what the ticked ones leave of the memory budget. The figure is what
// makes a pinned set that cannot fit visible while it is being chosen, rather
// than at the first request Dessau has to refuse.
function renderPinSwitches() {
  const box = $('pinList');
  const rows = pinRows(state.models || [], state.pinned || []);
  box.innerHTML = '';
  if (!rows.length) {
    box.innerHTML = '<p class="hint">Download a model and it appears here.</p>';
    updatePinBudget();
    return;
  }
  rows.forEach((r) => {
    const row = document.createElement('label');
    row.className = 'switch';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.dataset.model = r.id;
    cb.checked = r.checked;
    cb.addEventListener('change', () => { settingsTouched = true; updatePinBudget(); });
    const name = document.createElement('span');
    name.textContent = r.absent ? `${r.id} (not on this Mac)` : r.id;
    row.appendChild(cb);
    row.appendChild(name);
    box.appendChild(row);
  });
  updatePinBudget();
}

// pinRows is the whole list of boxes the form draws: one per model on this
// Mac, then one per pin that names a model this Mac does not have.
//
// The second half is not a nicety. The form carries through every pin it does
// not list (see modelSettings), so a pin with no box could never be removed —
// and a model deleted after it was pinned, or pinned before it was downloaded,
// leaves exactly that. Showing it is what makes every pin removable by the
// form that made it.
function pinRows(models, pinned) {
  const want = (pinned || []).map(foldRepoID);
  const have = new Set((models || []).map((m) => foldRepoID(m.repo_id)));
  const rows = (models || []).map((m) => ({
    id: m.repo_id,
    checked: want.includes(foldRepoID(m.repo_id)),
    absent: false,
  }));
  (pinned || []).filter((id) => !have.has(foldRepoID(id))).forEach((id) => {
    rows.push({ id, checked: true, absent: true });
  });
  return rows;
}

// pinBoxes are the boxes drawn above, one per model the form lists.
function pinBoxes() {
  return Array.from(document.querySelectorAll('#pinList input[type=checkbox]'));
}

// listedPinModels lists the models the form drew a box for.
function listedPinModels() {
  return pinBoxes().map((cb) => cb.dataset.model);
}

// checkedPinModels lists the models whose box is ticked.
function checkedPinModels() {
  return pinBoxes().filter((cb) => cb.checked).map((cb) => cb.dataset.model);
}

// servedContext is the window Dessau serves a model at, which is what it is
// charged for and what the gateway holds a request to: the operator's figure
// for that model, or the window the model itself declares when they have set
// none or set one the model cannot address. It is config.Config.ServedContext
// written out again here, and a test in internal/ui holds the two together.
function servedContext(config, repoID, declared) {
  const models = (config || {}).models || {};
  let set = 0;
  Object.keys(models).forEach((id) => {
    if (foldRepoID(id) === foldRepoID(repoID)) set = models[id].served_context || 0;
  });
  if (set <= 0 || (declared > 0 && set > declared)) return declared || 0;
  return set;
}

// modelCharge is what one model costs the memory budget, and it is the Go
// charge (capability.LoadCostOf) written out again here: the weights plus a
// fifth for the working set, plus the cache the window this model is served at
// will build, once per sequence its server may decode at once. A model whose
// configuration says nothing about its cache is charged the flat figure.
//
// No ceiling: a charge is what the model will cost. The safety factor is not
// here either — what the models list carries is already the charged cost per
// token, so the panel multiplies and nothing more. A test in internal/ui holds
// this to the Go figure; if you change one, change both.
function modelCharge(model, config, sequences) {
  const m = model || {};
  // A model still downloading is charged the size it declares, because ticking
  // its box now is a promise about the memory it will take when it lands.
  const bytes = m.bytes || m.size_bytes || 0;
  const flat = bytes + Math.floor(bytes / 5);
  const perToken = m.kv_charge_per_token || 0;
  const window = servedContext(config, m.repo_id, m.context_length || 0);
  const seq = sequences || 0;
  if (perToken <= 0 || window <= 0 || seq <= 0) return flat;
  return flat + perToken * window * seq;
}

// pinnedCharge is what the pinned models cost against the memory budget. A pin
// naming a model this Mac does not have at all has no size to charge.
function pinnedCharge(models, pinned, config, sequences) {
  const want = new Set((pinned || []).map(foldRepoID));
  return (models || []).reduce((sum, m) => {
    if (!want.has(foldRepoID(m.repo_id))) return sum;
    return sum + modelCharge(m, config, sequences);
  }, 0);
}

// budgetBytes is what the memory-budget field posts: the operator types
// gigabytes, the settings file stores bytes. A blank field — or anything that
// is not a positive number — is zero, which means the default share of this
// Mac's memory rather than a budget of nothing.
function budgetBytes(text) {
  const gb = parseFloat(text);
  if (!isFinite(gb) || gb <= 0) return 0;
  const bytes = Math.round(gb * 1024 * 1024 * 1024);
  // A figure no byte count can hold is not a budget. Left as Infinity it
  // serializes as null, which the server reads as "the field was not sent" and
  // answers by keeping what it had — a save that silently does nothing.
  if (!isFinite(bytes) || bytes > Number.MAX_SAFE_INTEGER) return 0;
  return bytes;
}

// budgetFieldValue is what the field shows: blank while the budget is the
// default, so that saving an unrelated setting does not quietly turn the
// default into a figure of the operator's own.
function budgetFieldValue(machine) {
  const m = machine || {};
  if (!m.budget || m.budget_is_default) return '';
  // Unrounded, so that what is shown posts back as the figure it came from, to
  // the byte: a budget set through the API is not always a round number of
  // gigabytes, and rounding it here would rewrite it on the next unrelated
  // save. Dividing by a power of two is exact, and JavaScript renders a number
  // as the shortest text that reads back as the same number, so the round trip
  // through budgetBytes is exact for any byte count a budget can hold. The
  // field is step="any" for the same reason: a stepped field refuses the whole
  // form for a value it does not land on, and the form holds every other
  // setting there is.
  return String(m.budget / (1024 * 1024 * 1024));
}

// budgetHint is the line beside the field: what the budget is, what share of
// this Mac that is, what the models in memory are using of it, and — above the
// warning threshold — that this memory is shared with everything else running.
//
// A Mac whose memory could not be read states the budget alone: a percentage
// of an unknown total is a number pretending to be information.
function budgetHint(machine) {
  const m = machine || {};
  const budget = m.budget || 0;
  const resident = m.resident_bytes || 0;
  const share = m.total_ram
    ? ` (${Math.round((budget * 100) / m.total_ram)}% of this Mac's ${size(m.total_ram)})`
    : ', because this Mac\'s memory could not be read';
  const parts = [`${m.budget_is_default ? 'The default budget is' : 'The budget is'} ${size(budget)}${share}.`];
  if (m.over_budget) {
    parts.push(`The models in memory use ${size(resident)}, over the budget: a lower budget applies to the next load, and nothing is unloaded on your behalf.`);
  } else if (resident) {
    parts.push(`The models in memory use ${size(resident)} of it.`);
  }
  if (m.warn_above && budget > m.warn_above) {
    // The second half is the server's own sentence, unescaped and unsplit so
    // that internal/ui/budget_test.go can hold it to app.BudgetChargeNote —
    // the server says the same thing when such a budget is saved.
    parts.push('macOS and everything else running share this memory, and '
      + "a model's charge is worked out from its configuration rather than measured on this Mac.");
  }
  return parts.join(' ');
}

// budgetShown is the machine as the figure being typed would leave it: what
// saving now would give. A cleared field is the default, so it shows the
// default rather than the figure it would replace — the panel documents
// clearing the field as the way back, and showing the old number there would
// make that instruction read as a lie.
function budgetShown(machine, typed) {
  const m = machine || {};
  const budget = typed || m.default_budget || m.budget || 0;
  return Object.assign({}, m, {
    budget,
    budget_is_default: !typed,
    over_budget: budget > 0 && (m.resident_bytes || 0) > budget,
  });
}

// updateBudgetHint writes that line, against the figure being typed rather
// than the one last saved, so the warning arrives while the number is being
// chosen instead of after it is stored.
function updateBudgetHint() {
  const line = $('budgetHint');
  if (!line) return;
  const shown = budgetShown(state.machine, budgetBytes($('setBudget').value));
  line.textContent = budgetHint(shown);
  line.className = shown.over_budget || (shown.warn_above && shown.budget > shown.warn_above) ? 'msg err' : 'hint';
}

// concurrencyNotice says when the batched-requests figure in the box is not
// the one Dessau is running with. The decode concurrency is a pool option,
// read once when the pool is built, so a save only reaches it at the next
// start — and until then the memory a model is charged, which is worked out
// once per sequence, follows the figure in force and not the one on screen.
// Nothing said so, and the pinned-charge line silently used the saved figure
// (iss-2609190021445846).
//
// Empty while the two agree, and while either is unknown: a notice about a
// difference nobody can see would be worse than none.
function concurrencyNotice(machine, config) {
  const inForce = (machine && machine.decode_concurrency) || 0;
  const saved = (config && config.decode_concurrency) || 0;
  if (!inForce || !saved || inForce === saved) return '';
  return `Dessau is batching ${inForce} request${inForce === 1 ? '' : 's'} at a time. `
    + `The saved figure of ${saved} takes effect at the next start, and the memory a `
    + `pinned model is charged below is worked out from the ${inForce} in force.`;
}

function updateConcurrencyNotice() {
  const line = $('concHint');
  if (!line) return;
  const text = concurrencyNotice(state.machine, state.config);
  line.textContent = text;
  line.hidden = text === '';
}

// updatePinBudget writes the line beside the boxes: what the ticked models
// cost, and what that leaves for everything else.
function updatePinBudget() {
  const line = $('pinBudget');
  if (!line) return;
  const budget = (state.machine && state.machine.budget) || 0;
  // The decode concurrency is part of the charge: each sequence a server may
  // run at once holds its own cache. Both figures come from the machine block,
  // which is what the pool is running with — not from state.config, whose
  // decode concurrency is the saved value and does not reach the pool until a
  // restart. Mixing the two gave a charge the pool would not agree with for as
  // long as a save was waiting for one (iss-2609190021445846); concurrencyNotice
  // says so beside the field.
  const sequences = (state.machine && state.machine.decode_concurrency) || 0;
  const charge = pinnedCharge(state.models || [], checkedPinModels(), state.config, sequences);
  if (!budget) {
    line.textContent = charge ? `Pinned models use about ${size(charge)}.` : '';
    line.className = 'hint';
    return;
  }
  const left = budget - charge;
  line.textContent = left >= 0
    ? `Pinned models use about ${size(charge)} of the ${size(budget)} memory budget, leaving ${size(left)} for everything else.`
    : `Pinned models use about ${size(charge)}, more than the ${size(budget)} memory budget — this cannot be saved.`;
  line.className = left >= 0 ? 'hint' : 'msg err';
}

// renderMergeSwitches draws one box per downloaded model. Merging is per model
// because it is a fact about that model's chat template, and because it is the
// one setting that has Dessau read a request's instructions at all.
function renderMergeSwitches() {
  const box = $('mergeList');
  const models = state.models || [];
  const per = state.config.models || {};
  box.innerHTML = '';
  if (!models.length) {
    box.innerHTML = '<p class="hint">Download a model and it appears here.</p>';
    return;
  }
  models.forEach((m) => {
    const row = document.createElement('label');
    row.className = 'switch';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.dataset.model = m.repo_id;
    cb.checked = !!(per[m.repo_id] && per[m.repo_id].merge_system_messages);
    cb.addEventListener('change', () => { settingsTouched = true; });
    const name = document.createElement('span');
    name.textContent = m.repo_id;
    row.appendChild(cb);
    row.appendChild(name);
    box.appendChild(row);
  });
}

// renderContextFields draws one window field per downloaded model. A blank
// field is the model's own declared window, which is what the placeholder
// shows, so the operator types a figure only for a model they want served
// shorter than it was built for.
function renderContextFields() {
  const box = $('contextList');
  if (!box) return;
  const models = state.models || [];
  const per = state.config.models || {};
  box.innerHTML = '';
  if (!models.length) {
    box.innerHTML = '<p class="hint">Download a model and it appears here.</p>';
    return;
  }
  models.forEach((m) => {
    const row = document.createElement('label');
    row.className = 'field';
    const name = document.createElement('span');
    name.textContent = m.repo_id;
    const input = document.createElement('input');
    input.type = 'number';
    input.min = '0';
    input.dataset.model = m.repo_id;
    input.placeholder = m.context_length ? String(m.context_length) : 'the model\'s own window';
    const set = per[m.repo_id] && per[m.repo_id].served_context;
    input.value = set ? String(set) : '';
    input.addEventListener('input', () => { settingsTouched = true; updatePinBudget(); });
    row.appendChild(name);
    row.appendChild(input);
    box.appendChild(row);
  });
}

// contextInputs are the fields drawn above, one per model the form lists.
function contextInputs() {
  return Array.from(document.querySelectorAll('#contextList input[type=number]'));
}

// listedContextModels lists the models the form drew a field for.
function listedContextModels() {
  return contextInputs().map((el) => el.dataset.model);
}

// typedContextModels is what has been typed into those fields, by model. A
// blank or unusable field is zero, which applyModelNumber reads as "the
// model's own window" and leaves no setting behind.
function typedContextModels() {
  const out = {};
  contextInputs().forEach((el) => { out[el.dataset.model] = parseInt(el.value, 10) || 0; });
  return out;
}

// mergeBoxes are the boxes drawn above, one per model the form lists.
function mergeBoxes() {
  return Array.from(document.querySelectorAll('#mergeList input[type=checkbox]'));
}

// listedMergeModels lists the models the form drew a box for.
function listedMergeModels() {
  return mergeBoxes().map((cb) => cb.dataset.model);
}

// checkedMergeModels lists the models whose box is ticked.
function checkedMergeModels() {
  return mergeBoxes().filter((cb) => cb.checked).map((cb) => cb.dataset.model);
}

// modelSettings returns the whole per-model map a save posts: one map holding
// every setting that belongs to a model rather than to the machine — merging,
// pinning and the sampling override.
//
// The server replaces what it holds with this, so a model whose box is clear
// is simply left out and the setting goes off — a map cannot be switched off
// by omission any other way. Two things are therefore carried through rather
// than rebuilt. A setting this form does not own stays on the model that has
// it, so ticking one box never wipes another setting. And a model the form
// does not list keeps everything it has, because a model can be given settings
// before it is downloaded and a form with no box for it has nothing to say
// about it.
//
// The sampling overrides are not a box but a whole editor, which holds every
// override there is while it is open, so they are assigned rather than
// toggled: a model missing from them has had its override removed.
function modelSettings(current, overrides, listedMerge, checkedMerge, listedPin, checkedPin, listedContext, typedContext) {
  const out = {};
  Object.keys(current || {}).forEach((id) => {
    out[id] = Object.assign({}, current[id]);
    delete out[id].sampling;
  });
  Object.keys(overrides || {}).forEach((id) => {
    out[id] = Object.assign({}, out[id] || {}, { sampling: overrides[id] });
  });
  applyModelSwitch(out, 'merge_system_messages', listedMerge, checkedMerge);
  applyModelSwitch(out, 'pinned', listedPin, checkedPin);
  applyModelNumber(out, 'served_context', listedContext, typedContext);
  // A model left with no settings at all is left out entirely, so that
  // clearing every box for a model removes it rather than storing an empty
  // object under its name.
  Object.keys(out).forEach((id) => {
    if (!Object.keys(out[id]).length) delete out[id];
  });
  return out;
}

// applyModelSwitch writes one row of boxes into the map being posted: every
// model the form drew a box for loses the setting, and every model whose box
// is ticked gets it back. A model with no box is not touched.
function applyModelNumber(models, field, listed, typed) {
  (listed || []).forEach((id) => {
    if (models[id]) delete models[id][field];
  });
  Object.keys(typed || {}).forEach((id) => {
    const n = typed[id];
    // A blank or unusable field is the model's own window, which is the
    // absence of the setting rather than a figure of zero.
    if (!(n > 0)) return;
    models[id] = Object.assign({}, models[id] || {}, { [field]: n });
  });
}

function applyModelSwitch(models, field, listed, checked) {
  (listed || []).forEach((id) => {
    if (models[id]) delete models[id][field];
  });
  (checked || []).forEach((id) => {
    models[id] = Object.assign({}, models[id] || {}, { [field]: true });
  });
}

function renderOverrides() {
  renderOverrideList();
  refreshOverrideModels();
  prefillOverride();
}

function renderOverrideList() {
  const list = $('overrideList');
  list.innerHTML = '';
  Object.keys(overrides).sort().forEach((id) => {
    const row = document.createElement('div');
    row.className = 'override';
    const meta = document.createElement('div');
    const name = document.createElement('div');
    name.className = 'name';
    name.textContent = id;
    const values = document.createElement('div');
    values.className = 'values';
    values.textContent = describeSampling(overrides[id]) || 'no parameters set';
    meta.append(name, values);
    row.append(meta, btn('Remove', 'ghost', () => {
      delete overrides[id];
      settingsTouched = true;
      renderOverrides();
    }));
    list.appendChild(row);
  });

}

// refreshOverrideModels rebuilds the list of models that can be given an
// override, keeping the current selection. It runs on every state frame, not
// only when the form is redrawn, so a model that finishes downloading while
// the form is being edited can be chosen without reloading the page.
function refreshOverrideModels() {
  // Offer every model on this Mac, plus any model an override already names
  // (one whose files have since been deleted still has a saved override).
  const select = $('ovModel');
  const chosen = select.value;
  const ids = new Set((state.models || []).map((m) => m.repo_id));
  Object.keys(overrides).forEach((id) => ids.add(id));
  select.innerHTML = '';
  [...ids].sort().forEach((id) => {
    const opt = document.createElement('option');
    opt.value = id;
    opt.textContent = id;
    select.appendChild(opt);
  });
  if (chosen && ids.has(chosen)) select.value = chosen;
}

// prefillOverride shows what is stored for the selected model. Setting an
// override replaces it whole, so the fields have to start from the saved
// values — otherwise changing a temperature quietly drops the token budget
// saved beside it.
function prefillOverride() {
  writeSampling('over', overrides[$('ovModel').value]);
}

// foldPendingOverride takes whatever is in the per-model fields and makes it
// the selected model's override; every field blank removes it. Called both
// from the button and from the form's submit, because the fields sit inside
// the settings form and pressing Save (or Enter in a number field) must not
// throw away what is typed in them.
function foldPendingOverride() {
  const id = $('ovModel').value;
  if (!id) return;
  const values = readSampling('over');
  if (isEmptySampling(values)) {
    delete overrides[id];
  } else {
    overrides[id] = values;
  }
}

$('ovModel').addEventListener('change', prefillOverride);

$('ovApply').addEventListener('click', () => {
  foldPendingOverride();
  settingsTouched = true;
  renderOverrides();
});

$('setContextProbe').addEventListener('change', () => { settingsTouched = true; });
$('setIdleThreshold').addEventListener('input', () => { settingsTouched = true; });
$('setSelfTest').addEventListener('change', () => { settingsTouched = true; });
$('setStats').addEventListener('change', () => { settingsTouched = true; });
$('setGrace').addEventListener('change', () => { settingsTouched = true; });
$('setAdvertise').addEventListener('change', () => { settingsTouched = true; });
$('setGraceSec').addEventListener('input', updateGraceHint);
$('setGraceWait').addEventListener('input', updateGraceHint);
$('setBudget').addEventListener('input', updateBudgetHint);

// Clear is not part of saving the form: it throws away what was recorded, so
// it happens when it is pressed and says what it did.
$('statsClear').addEventListener('click', async () => {
  if (!confirm('Remove every request record kept on this Mac? This cannot be undone. The recording switch is left as it is.')) return;
  try {
    await api('/api/stats/clear', { method: 'POST' });
    // Say so at once rather than at the next snapshot, so the button visibly
    // did something.
    if (state) state.stats_store = null;
    renderStatsStore();
    refreshStats();
  } catch (err) {
    $('settingsMsg').className = 'msg err';
    $('settingsMsg').textContent = err.message;
  }
});

// renderStatsStore says what is actually kept on disk, which is what makes the
// two figures above it mean something: a limit in megabytes says nothing about
// whether the store holds a week or a year.
function renderStatsStore() {
  const line = $('statsStoreLine');
  const store = state && state.stats_store;
  if (!store) {
    line.textContent = '';
    return;
  }
  if (store.refused) {
    line.textContent = 'Dessau could not open the store where records are kept, so the figures ' +
      'above are being held in memory only and nothing is on disk. Its own log says why.';
    return;
  }
  const parts = [];
  if (store.oldest) {
    parts.push(`Records from ${new Date(store.oldest * 1000).toLocaleDateString()} onwards`);
  } else {
    parts.push('No records kept yet');
  }
  parts.push(`${(store.bytes / (1024 * 1024)).toFixed(1)} MB in ${store.files} file${store.files === 1 ? '' : 's'}`);
  if (store.retention_wedged) {
    parts.push('what is being dropped cannot be summarised first — the records are kept while there is ' +
      'room for them, and once there is not the oldest go without a summary (its own log says why)');
  }
  if (store.unsummarized) {
    parts.push(`${store.unsummarized} removed without a summary`);
  }
  if (store.stalled) {
    parts.push('the disk did not answer in time, so these figures may not include the newest records ' +
      '(its own log says so, and they come back as soon as it does)');
  }
  if (store.dropped) {
    parts.push(`${store.dropped} not written — the disk could not keep up`);
  }
  if (store.skipped) {
    parts.push(`${store.skipped} unreadable lines`);
  }
  line.textContent = `${parts.join(' · ')}.`;
}

// chatRule turns the two settings fields into the rule the server holds: two
// lists of HuggingFace's own words, as typed, trimmed, with the blanks a comma
// or two leaves behind dropped.
//
// A cleared field posts an empty list rather than nothing at all. The two are
// different settings on the server — an absent rule means the shipped default,
// an empty list means "do not test this half" — and a cleared field is the
// operator asking for the second.
function chatRule(pipelines, tags) {
  return { pipeline_tags: splitTagField(pipelines), required_tags: splitTagField(tags) };
}

function splitTagField(value) {
  return String(value || '')
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s !== '');
}

$('genKey').addEventListener('click', () => {
  const b = new Uint8Array(24);
  crypto.getRandomValues(b);
  const hex = Array.from(b).map((x) => x.toString(16).padStart(2, '0')).join('');
  $('setKey').value = `bh_${hex}`;
  settingsTouched = true;
});

$('settingsForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const msg = $('settingsMsg');
  // Anything typed in the per-model fields counts, whether or not Set override
  // was pressed.
  foldPendingOverride();
  // Only the fields this form owns. Anything omitted — the preload list, the
  // upstream header timeout — is preserved server-side, which is what keeps a
  // save from dropping a setting the operator set by hand and never touched
  // here. Advertising used to be one of those: omitting it preserved it, and
  // an earlier form that posted advertise:true silently re-enabled the advert
  // on every save. It is a control now, so it is posted from the box.
  const body = {
    // The bind is two fields — an address and a mode — and the select carries
    // whichever one the operator chose.
    ...bindSelectBody($('setHost').value, state.config.host),
    port:               parseInt($('setPort').value, 10),
    tls_port:           parseInt($('setTLSPort').value, 10),
    advertise:          $('setAdvertise').checked,
    api_key:            $('setKey').value,
    idle_timeout_sec:   parseInt($('setIdle').value, 10) || 0,
    decode_concurrency: parseInt($('setConc').value, 10) || 1,
    hf_token:           $('setHF').value,
    discord_bridge:     $('setDiscordBridge').checked,
    discord_token:      $('setDiscordToken').value,
    max_resident_bytes: budgetBytes($('setBudget').value),
    eviction_grace:        $('setGrace').checked,
    // Zero is what the server reads as "the default", which is what a cleared
    // field means here, so an unparseable or empty box posts zero rather than
    // a figure nobody typed.
    eviction_grace_sec:    parseInt($('setGraceSec').value, 10) || 0,
    eviction_max_wait_sec: parseInt($('setGraceWait').value, 10) || 0,
    chat_rule:          chatRule($('setChatPipelines').value, $('setChatTags').value),
    context_probe:      $('setContextProbe').checked,
    // Blank posts zero, which the server reads as the default.
    idle_threshold_sec: parseInt($('setIdleThreshold').value, 10) || 0,
    self_test:          $('setSelfTest').checked,
    statistics:         $('setStats').checked,
    stats_months:       parseInt($('setStatsMonths').value, 10) || 6,
    stats_max_bytes:    (parseInt($('setStatsMB').value, 10) || 200) * 1024 * 1024,
    log_level:          $('setLogLevel').value,
    // A blank sampling field is sent as null, not as zero: the model server is
    // handed a flag only for a parameter that has a value.
    sampling:           readSampling('input'),
    models: modelSettings(
      state.config.models, overrides,
      listedMergeModels(), checkedMergeModels(),
      listedPinModels(), checkedPinModels(),
      listedContextModels(), typedContextModels(),
    ),
  };
  try {
    const res = await api('/api/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    msg.className = 'msg';
    const parts = ['Saved.'];
    if (res.restart) parts.push('Restart Dessau for the change to take effect.');
    // Sampling defaults are set when a model server starts, so a model that is
    // already loaded keeps the values it started with.
    if (res.warning) parts.push(res.warning);
    if (res.reload_models && res.reload_models.length) {
      parts.push(`Load ${res.reload_models.join(', ')} again to serve with the new sampling defaults.`);
    }
    msg.textContent = parts.join(' ');
    settingsTouched = false;
  } catch (err) {
    msg.className = 'msg err';
    msg.textContent = err.message;
  }
});

// ── statistics ───────────────────────────────────────────
// Nothing here is shown, and nothing is fetched, until the operator turns
// recording on: with the switch off the endpoint answers with an empty view
// and the panel says so.

let statsTimer = null;

// watchStats starts or stops the polling that keeps the Statistics view fresh
// while it is the visible one.
function watchStats(visible) {
  if (statsTimer) { clearInterval(statsTimer); statsTimer = null; }
  if (!visible) return;
  refreshStats();
  // The historical tables are fetched when the view opens and when the range
  // changes, and never on this tick: they are a pass over months of records,
  // and asking for one every two seconds would spend the Mac's afternoon
  // redrawing a table nobody is watching change.
  refreshHistory();
  statsTimer = setInterval(refreshStats, 2000);
}

async function refreshStats() {
  try {
    renderStats(await api('/api/stats'));
  } catch {
    // A panel that cannot reach its own server already says "disconnected" at
    // the top; a second alert about it would be noise.
  }
  // On the same tick, never a timer of its own: the self-test's file changes
  // at most once a run, and one cadence for the tab is one cadence to reason
  // about.
  try {
    renderSelfTest(await api('/api/selftest'));
  } catch {
    // As above.
  }
}

// selfTestRow flattens one run into the figures the table shows, each a
// string ready to print, so the row is a pure function a test can hold.
function selfTestRow(run) {
  const tests = {};
  for (const t of run.tests || []) tests[t.name] = t;
  const pp = tests.pp512;
  const tg = tests.tg128;
  const batch = Object.values(tests).find((t) => t.parallel > 1);
  const outcome = run.outcome === 'ok' ? 'ok'
    : run.outcome === 'yielded' ? 'yielded to a request'
      : run.outcome === 'stopped' ? 'stopped'
        : `failed (${run.reason || 'unknown'})`;
  return {
    model: run.model,
    measured: new Date(run.at * 1000).toLocaleString(),
    outcome,
    load: run.cold_load ? `${(run.load_ms / 1000).toFixed(1)} s` : 'already loaded',
    promptRead: pp ? `${pp.prompt_tokens_per_sec} tok/s` : '',
    firstToken: tg ? `${tg.first_token_ms} ms` : '',
    generation: tg ? `${tg.tokens_per_sec} tok/s` : '',
    underLoad: batch ? `${batch.tokens_per_sec} tok/s ×${batch.parallel}` : '',
  };
}

// selfTestHint says whether the self-test is running and, while it is, how
// long this Mac must go unasked before the next run — in the words the panel
// was given rather than a figure it holds. A test that is off runs at no
// interval, so the off lines name none.
function selfTestHint(on, measured, idleFor) {
  if (!on) {
    return measured
      ? 'Off. These are the runs from when it was on; turn it on in Settings to measure again.'
      : 'Off. Turn on Test the models when this Mac is idle in Settings to measure your models.';
  }
  return measured
    ? `On. Dessau measures a model whenever this Mac has been idle for ${idleFor}.`
    : `On. Nothing measured yet: the first run starts once this Mac has been idle for ${idleFor}.`;
}

function renderSelfTest(view) {
  const on = !!(view && view.enabled);
  const latest = (view && view.latest) || [];
  $('selftestHint').textContent = selfTestHint(on, latest.length > 0, idleThresholdWords(state));
  $('selftestBody').hidden = latest.length === 0;
  const rows = $('selftestRows');
  rows.replaceChildren();
  for (const run of latest) {
    const r = selfTestRow(run);
    const tr = document.createElement('tr');
    for (const cell of [r.model, r.measured, r.outcome, r.load, r.promptRead, r.firstToken, r.generation, r.underLoad]) {
      const td = document.createElement('td');
      td.textContent = cell;
      tr.appendChild(td);
    }
    rows.appendChild(tr);
  }
}

function renderStats(view) {
  const on = !!(view && view.enabled);
  $('statsOff').hidden = on;
  $('statsBody').hidden = !on;
  if (!on) return;

  const totals = recentTotals(view.rollups || [], Math.floor(Date.now() / 1000));
  $('statsTotals').textContent =
    `Last hour: ${totals.hourRequests} request${totals.hourRequests === 1 ? '' : 's'}, ` +
    `${totals.hourCompletion} tokens generated · ` +
    `Last 24 hours: ${totals.dayRequests} request${totals.dayRequests === 1 ? '' : 's'}, ` +
    `${totals.dayCompletion} tokens generated`;

  $('statsModels').innerHTML = (view.models || []).length
    ? (view.models || []).map(modelStatsCard).join('')
    : '<p class="hint">No requests yet.</p>';

  // Newest first: the request a reader wants is almost always the last one.
  const rows = (view.requests || []).slice().reverse();
  $('statsRows').innerHTML = rows.map((r) => {
    const c = requestRow(r);
    const bad = c.failed ? ' bad' : '';
    return `<tr>
      <td>${c.time}</td>
      <td>${escapeHtml(c.model)}</td>
      <td>${c.mode}</td>
      <td class="${c.failed ? 'bad' : ''}">${c.outcome}</td>
      <td class="figure${bad}">${c.tokens}</td>
      <td class="figure${bad}">${c.first}</td>
      <td class="figure${bad}">${c.rate}</td>
      <td class="figure${bad}">${c.waited}</td>
      <td class="figure${bad}">${c.total}</td>
    </tr>`;
  }).join('');

  renderStatsSummary(view.summaries || []);
}

// renderStatsSummary shows what is left of the days the store no longer holds
// in detail. The table above it reaches back only as far as retention allowed;
// without this, a month whose records have been dropped looks like a month with
// no traffic in it.
function renderStatsSummary(days) {
  const block = $('statsSummaryBlock');
  block.hidden = days.length === 0;
  if (!days.length) return;
  const gone = days.filter((d) => !d.detail_held).length;
  $('statsSummaryNote').textContent = gone
    ? `The detailed records for ${gone === days.length ? 'these days' : `${gone} of these days`} are no longer held — ` +
      'these totals are all that is kept of them. Days marked "partly kept" have some records left in the table above, ' +
      'and these totals cover only the part that was dropped.'
    : 'These totals cover the records already dropped for these days; the rest are still in the table above.';
  $('statsSummaryRows').innerHTML = days.map((d) => {
    const first = d.first_token_requests
      ? millis(Math.round(d.first_token_ms_total / d.first_token_requests))
      : '—';
    const total = d.requests ? millis(Math.round(d.duration_ms_total / d.requests)) : '—';
    return `<tr>
      <td>${escapeHtml(d.day || '—')}</td>
      <td>${escapeHtml(d.model || '—')}</td>
      <td class="figure">${d.requests}</td>
      <td class="figure">${d.prompt_tokens} / ${d.completion_tokens}</td>
      <td class="figure">${first}</td>
      <td class="figure">${total}</td>
      <td>${d.detail_held ? 'partly kept' : 'gone'}</td>
    </tr>`;
  }).join('');
}

// recentTotals adds the minute buckets up over the last hour and the last day.
// The buckets are the only thing that reaches back further than the thousand
// rows the ring holds, which at a busy minute is not far at all.
function recentTotals(rollups, nowSeconds) {
  const hourFrom = nowSeconds - 3600;
  const dayFrom = nowSeconds - 86400;
  const t = { hourRequests: 0, hourCompletion: 0, dayRequests: 0, dayCompletion: 0 };
  rollups.forEach((b) => {
    if (b.minute < dayFrom) return;
    t.dayRequests += b.requests;
    t.dayCompletion += b.completion_tokens;
    if (b.minute < hourFrom) return;
    t.hourRequests += b.requests;
    t.hourCompletion += b.completion_tokens;
  });
  return t;
}

// modelStatsCard is one model's totals since recording was turned on.
function modelStatsCard(m) {
  const failed = (m.requests || 0) - ((m.by_class || {}).ok || 0);
  const last = generationRate({
    class: 'ok',
    completion_tokens: m.last_completion_tokens,
    first_token_ms: m.last_first_token_ms,
    duration_ms: m.last_duration_ms,
  });
  const figures = [
    `${m.requests} request${m.requests === 1 ? '' : 's'}`,
    failed ? `${failed} did not answer` : null,
    `${m.prompt_tokens} tokens in · ${m.completion_tokens} out`,
    m.last_first_token_ms >= 0 ? `last first token ${millis(m.last_first_token_ms)}` : null,
    last !== '—' ? `last rate ${last}` : null,
    m.last_duration_ms ? `last request ${millis(m.last_duration_ms)}` : null,
    m.loads ? `loaded ${m.loads}×` : null,
    m.failed_loads ? `${m.failed_loads} failed to load` : null,
    m.evictions ? `evicted ${m.evictions}×` : null,
    m.last_load_ms ? `last load ${millis(m.last_load_ms)}` : null,
  ].filter(Boolean).join(' · ');
  return `<div class="statcard"><div class="name">${escapeHtml(m.model || '—')}</div>` +
    `<div class="figures">${figures}</div></div>`;
}

// requestRow is the whole of one row of the request table: a pure function of
// one record, so what a reader sees can be asserted without a DOM (see
// stats_test.go). Every cell is a figure, a fixed outcome word, or the id of a
// model this Mac holds — there is nothing else in a record to show.
function requestRow(r) {
  const answered = r.class === 'ok';
  const d = new Date(r.at * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  // Both waits are a wait for the model rather than for the answer, so they
  // are one cell: a reader wants to know how much of the total was spent
  // before generation began, not which queue it was spent in.
  const waited = (r.queue_wait_ms || 0) + (r.load_wait_ms || 0);
  return {
    time: `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`,
    model: r.model || '—',
    mode: r.streamed ? 'stream' : 'once',
    outcome: outcomeLabel(r.class),
    failed: !answered,
    tokens: answered ? `${r.prompt_tokens} / ${r.completion_tokens}` : '—',
    first: r.first_token_ms >= 0 ? millis(r.first_token_ms) : '—',
    rate: generationRate(r),
    waited: waited ? millis(waited) : '—',
    total: millis(r.duration_ms),
  };
}

// outcomeLabel says in words how a request ended. The recorder's own names are
// for the record and for whatever reads it later; a reader of the panel should
// not have to learn them.
function outcomeLabel(c) {
  return {
    ok: 'answered',
    client_error: 'rejected',
    upstream_status: 'model refused',
    busy: 'too busy',
    refused: 'no room',
    launch_failed: 'could not start',
    not_ready: 'never ready',
    unreachable: 'no answer',
    cancelled: 'client left',
    gateway_error: 'Dessau failed',
  }[c] || 'unknown';
}

// generationRate is the tokens after the first over the time spent generating
// them, which is what vLLM, the OpenTelemetry conventions and LM Studio all
// mean by tokens per second. A request with no first token, or with one token,
// has no rate at all rather than a figure that means something else.
function generationRate(r) {
  if (r.class !== 'ok' || r.first_token_ms < 0 || r.completion_tokens < 2) return '—';
  const generating = (r.duration_ms - r.first_token_ms) / 1000;
  if (generating <= 0) return '—';
  return `${((r.completion_tokens - 1) / generating).toFixed(1)} tok/s`;
}

// millis renders a duration the way a reader reads one: milliseconds while
// they are small enough to matter, seconds once they are not.
function millis(ms) {
  return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(1)} s`;
}

// ── the historical views ─────────────────────────────────
// Four tables over a chosen range, read from the records kept on this Mac:
// tokens per day by model with each model's share, how long requests took,
// the spread of those times, and when models were evicted and started. Everything shown is a sum, a
// count or the id of a model this Mac holds — the browser is handed the
// aggregate, never the records it was worked out from.

// historyDays is the chosen range; 0 means everything the store still keeps,
// which the server narrows to the widest range one view covers.
let historyDays = 30;

// historyAsked counts the readings asked for, so a slow answer to a range the
// reader has already moved on from is dropped rather than drawn under the new
// selector value. There is no debounce: flipping the selector is meant to be
// quick, and three flips are three readings, of which only the last is drawn.
let historyAsked = 0;

// maxHistoryDays is the widest range one reading covers, which is what
// "Everything kept" asks for. Sending it rather than the epoch is what keeps
// the answer from reporting that it narrowed a range nobody meant to be wider.
const maxHistoryDays = 366;

$('statsRange').addEventListener('change', () => {
  historyDays = parseInt($('statsRange').value, 10) || 0;
  refreshHistory();
});

async function refreshHistory() {
  const to = Math.floor(Date.now() / 1000);
  const days = historyDays > 0 ? historyDays : maxHistoryDays;
  const asked = ++historyAsked;
  let res;
  try {
    res = await fetch(`/api/stats/history?from=${to - days * 86400}&to=${to}`);
  } catch {
    // A panel that cannot reach its own server already says "disconnected" at
    // the top; a second alert about it would be noise.
    return;
  }
  if (asked !== historyAsked) return;
  // Two readings of the records run at once and no more, so a third is
  // refused. That is not a disconnection and the panel must not draw the
  // previous range's tables under the new selector value as though it were.
  if (res.status === 503) {
    showHistoryBusy(res.headers.get('Retry-After'));
    return;
  }
  let body = null;
  try { body = JSON.parse(await res.text()); } catch { /* not json */ }
  if (!res.ok || asked !== historyAsked) return;
  renderHistory(body);
}

// showHistoryBusy says why there are no tables, and for how long.
function showHistoryBusy(retryAfter) {
  $('statsHistoryBody').hidden = true;
  $('statsHistoryEmpty').hidden = true;
  $('statsHistoryBusy').hidden = false;
  $('statsHistoryBusy').textContent = busyLine(retryAfter);
}

// busyLine is what the reader is told when a reading is refused. The wait comes
// from the server's own Retry-After rather than from a guess repeated here.
function busyLine(retryAfter) {
  const secs = parseInt(retryAfter, 10);
  const wait = secs > 0 ? ` Try again in about ${secs} second${secs === 1 ? '' : 's'}.` : '';
  return `The records are already being read.${wait}`;
}

function renderHistory(h) {
  if (!h) return;
  const days = h.days || [];
  // Emptiness is judged on every table, not on the requests alone: a range
  // that holds only loads and evictions has an hourly table with figures in
  // it, and saying "nothing was recorded" over the top of it would deny the
  // records it is drawn from.
  const anything = days.length > 0 || (h.latency || []).length > 0
    || (h.hours || []).some((x) => x.evictions || x.loads)
    || (h.prompt_sizes || []).length > 0 || (h.footprints || []).length > 0;
  $('statsHistoryBusy').hidden = true;
  $('statsHistoryBody').hidden = !anything;
  $('statsHistoryEmpty').hidden = anything;
  $('statsHistoryBounds').textContent = historyBoundsLine(h);

  // The share is the model's over the whole range, so it is looked up per row
  // rather than worked out from the day the row is on.
  const share = {};
  (h.models || []).forEach((m) => { share[m.model] = m.share; });
  $('statsDaysRows').innerHTML = days.map((d) => dayRowHtml(d, share[d.model])).join('');

  const latency = h.latency || [];
  $('statsLatencyRows').innerHTML = latency.map(latencyRowHtml).join('');
  const heads = bucketLabels(h.first_token_bucket_edges_ms || [])
    .map((l) => `<th>${escapeHtml(l)}</th>`).join('');
  $('statsSpreadHead').innerHTML = `<tr><th>Model</th>${heads}</tr>`;
  $('statsSpreadRows').innerHTML = latency.map(spreadRowHtml).join('');

  $('statsHoursRows').innerHTML = (h.hours || []).map(hourRowHtml).join('');

  $('statsSizesRows').innerHTML = (h.prompt_sizes || []).map(sizesRowHtml).join('');
  $('statsOverridesRows').innerHTML = (h.overrides || []).map(overridesRowHtml).join('');
  $('statsFootprintRows').innerHTML = (h.footprints || []).map(footprintRowHtml).join('');
}

// sizesRow is one model's prompts against its served window: the two
// windows, the count, the four bands and the refused, the largest seen. A
// pure function a test holds.
function sizesRow(p) {
  const b = p.buckets || [0, 0, 0, 0, 0];
  return {
    model: p.model || '—',
    declared: p.declared_context ? tokensLabel(p.declared_context) : '—',
    served: p.served_context ? tokensLabel(p.served_context) : '—',
    requests: p.requests || 0,
    bands: [b[0] || 0, b[1] || 0, b[2] || 0, b[3] || 0],
    refused: p.refused_for_size || 0,
    largest: p.largest_estimate ? tokensLabel(p.largest_estimate) : '—',
  };
}

function sizesRowHtml(p) {
  const r = sizesRow(p);
  return `<tr>
      <td>${escapeHtml(r.model)}</td>
      <td class="figure">${escapeHtml(r.declared)}</td>
      <td class="figure">${escapeHtml(r.served)}</td>
      <td class="figure">${figure(r.requests)}</td>
      ${r.bands.map((n) => `<td class="figure">${figure(n)}</td>`).join('')}
      <td class="figure">${figure(r.refused)}</td>
      <td class="figure">${escapeHtml(r.largest)}</td>
    </tr>`;
}

// overridesRow is how often one model's clients set each parameter, as a
// share of its requests.
function overridesRow(o) {
  const by = o.by_parameter || {};
  const n = o.requests || 0;
  const pct = (k) => (n > 0 ? `${Math.round(((by[k] || 0) * 100) / n)}%` : '—');
  return {
    model: o.model || '—', requests: n,
    temperature: pct('temperature'), top_p: pct('top_p'), top_k: pct('top_k'), min_p: pct('min_p'), max_tokens: pct('max_tokens'),
  };
}

function overridesRowHtml(o) {
  const r = overridesRow(o);
  return `<tr>
      <td>${escapeHtml(r.model)}</td>
      <td class="figure">${figure(r.requests)}</td>
      <td class="figure">${r.temperature}</td>
      <td class="figure">${r.top_p}</td>
      <td class="figure">${r.top_k}</td>
      <td class="figure">${r.min_p}</td>
      <td class="figure">${r.max_tokens}</td>
    </tr>`;
}

// sparkline draws a series as one character per point, eight heights from
// the lowest to the highest, so the line over the range is on the page and
// not only in the figures.
function sparkline(points) {
  const bars = '▁▂▃▄▅▆▇█';
  const pts = points || [];
  if (pts.length === 0) return '';
  let lo = Infinity; let hi = 0;
  pts.forEach((p) => { if (p.bytes < lo) lo = p.bytes; if (p.bytes > hi) hi = p.bytes; });
  const span = hi - lo;
  return pts.map((p) => bars[span > 0 ? Math.min(7, Math.floor(((p.bytes - lo) * 8) / span)) : 4]).join('');
}

// footprintRow folds one model's series into the figures a table can carry:
// how many points, the lowest, the highest, the latest, and the line itself.
function footprintRow(f) {
  const pts = f.points || [];
  let lo = 0; let hi = 0;
  pts.forEach((p) => {
    if (!lo || p.bytes < lo) lo = p.bytes;
    if (p.bytes > hi) hi = p.bytes;
  });
  return {
    model: f.model || '—', samples: pts.length,
    lowest: pts.length ? bytes(lo) : '—', highest: pts.length ? bytes(hi) : '—',
    latest: pts.length ? bytes(pts[pts.length - 1].bytes) : '—',
    line: sparkline(pts),
  };
}

function footprintRowHtml(f) {
  const r = footprintRow(f);
  return `<tr>
      <td>${escapeHtml(r.model)}</td>
      <td class="figure">${figure(r.samples)}</td>
      <td class="figure">${escapeHtml(r.lowest)}</td>
      <td class="figure">${escapeHtml(r.highest)}</td>
      <td class="figure">${escapeHtml(r.latest)}</td>
      <td class="spark">${escapeHtml(r.line)}</td>
    </tr>`;
}

// historyBoundsLine says what the figures cover and what they were held to,
// because a table that quietly stopped short is worse than one that says it
// did: a reader would take a bounded month for a quiet one.
function historyBoundsLine(h) {
  // With recording off the answer is an empty one, and a range of 1 January
  // 1970 to 1 January 1970 is not a fact about anything.
  if (!h || !h.from || !h.to) return 'Nothing is recorded, so there is nothing to show over time.';
  const day = (secs) => new Date(secs * 1000).toLocaleDateString();
  const zone = h.zone ? ` (${h.zone})` : '';
  const parts = [`${day(h.from)} to ${day(h.to)}, by this Mac's own days and hours${zone}`];
  if (h.narrowed) {
    parts.push(`narrowed to the ${h.max_days} days one view covers`);
  }
  // A reading that met a bound after it had already read past the start of the
  // range covered the range: the bound stopped the walk beyond it, not the
  // figures. Only the other case is a figure drawn short.
  if (h.truncated && !h.reached_start) {
    parts.push(`${stopReason(h)}, short of the start of this range`);
  } else if (!h.reached_start) {
    parts.push('the records do not reach the start of this range');
  }
  if (h.skipped) {
    parts.push(`${h.skipped} lines could not be read`);
  }
  parts.push('nothing recorded while the switch was off appears here');
  return parts.join(' · ');
}

// stopReason names the bound that actually stopped the reading, with its own
// figure. Saying "a million records" when what stopped it was twenty thousand
// day rows is two orders of magnitude wrong in the one line whose purpose is
// that a table which stopped short does not read as a quiet month.
function stopReason(h) {
  switch (h.stopped_by) {
    case 'rows':    return `stopped after ${h.max_rows} rows of the table`;
    case 'bytes':   return `stopped after reading ${Math.round(h.max_bytes / (1024 * 1024))} MB of records`;
    case 'lines':   return `stopped after ${h.max_records} lines of the records`;
    case 'records': return `stopped after ${h.max_records} records`;
    default:        return 'stopped before the whole range was read';
  }
}

// dayRowHtml is one model's day: its own tokens, and its share of the whole
// range beside them, which is the figure that answers "which model does the
// work".
function dayRowHtml(d, share) {
  const total = (d.prompt_tokens || 0) + (d.completion_tokens || 0);
  // A day whose own records the retention has dropped keeps only the coarse
  // daily total, and a table that mixed the two without saying so would be
  // read as exact throughout.
  const coarse = d.from_summary ? ' <span class="pill">from daily totals</span>' : '';
  // The oldest day of a range that starts at four in the afternoon holds only
  // the traffic after four. Drawn beside whole days it reads as a quiet one, so
  // it says what it is.
  const clipped = d.partial ? ' <span class="pill">part of the day</span>' : '';
  return `<tr>
      <td>${escapeHtml(d.day)}</td>
      <td>${escapeHtml(d.model || '—')}${coarse}${clipped}</td>
      <td class="figure">${figure(d.requests)}</td>
      <td class="figure">${figure(d.prompt_tokens)}</td>
      <td class="figure">${figure(d.completion_tokens)}</td>
      <td class="figure">${figure(total)}</td>
      <td class="figure">${sharePercent(share)}</td>
    </tr>`;
}

// latencyRowHtml is one model's distribution: the middle request, the slow
// one, and the slowest but one.
function latencyRowHtml(l) {
  const first = l.first_token_ms || {};
  const rate = l.rate || {};
  return `<tr>
      <td>${escapeHtml(l.model || '—')}</td>
      <td class="figure">${figure(l.requests)}</td>
      <td class="figure">${msFigure(first.p50)}</td>
      <td class="figure">${msFigure(first.p90)}</td>
      <td class="figure">${msFigure(first.p99)}</td>
      <td class="figure">${figure(l.rate_requests)}</td>
      <td class="figure">${rateFigure(rate.p50)}</td>
      <td class="figure">${rateFigure(rate.p90)}</td>
      <td class="figure">${rateFigure(rate.p99)}</td>
    </tr>`;
}

// spreadRowHtml is the histogram as a row of counts, which is what a table can
// show of a distribution without drawing anything.
function spreadRowHtml(l) {
  const cells = (l.first_token_buckets || [])
    .map((n) => `<td class="figure">${figure(n)}</td>`).join('');
  return `<tr><td>${escapeHtml(l.model || '—')}</td>${cells}</tr>`;
}

// hourRowHtml is one hour of the Mac's own day.
function hourRowHtml(h) {
  const hour = String(h.hour).padStart(2, '0');
  return `<tr><td>${hour}:00</td><td class="figure">${figure(h.evictions)}</td>` +
    `<td class="figure">${figure(h.loads)}</td></tr>`;
}

// bucketLabels turns the histogram's edges into its column headings. The
// edges come from the server with the counts, so the headings cannot drift
// from the buckets they sit over.
function bucketLabels(edges) {
  if (!edges.length) return [];
  const label = (ms) => (ms < 1000 ? `${ms} ms` : `${ms / 1000} s`);
  const out = [`under ${label(edges[0])}`];
  for (let i = 1; i < edges.length; i += 1) {
    out.push(`${label(edges[i - 1])}–${label(edges[i])}`);
  }
  out.push(`${label(edges[edges.length - 1])} and over`);
  return out;
}

// sharePercent is a fraction as the panel shows it. The server sends the
// fraction rather than the percentage, so the rounding happens once, here.
function sharePercent(share) {
  if (!(share > 0)) return '—';
  return `${(share * 100).toFixed(1)}%`;
}

// figure is a count on its way into a cell. Every one of these is a number the
// server marshalled from a typed field, so nothing here can be markup — and
// this is the one place that has to stay true of, rather than a rule each cell
// is trusted to keep.
function figure(n) {
  return Number.isFinite(n) ? String(n) : '—';
}

// msFigure and rateFigure are a percentile as a reader reads one. A model with
// nothing to distribute has no figure at all; a model whose median first token
// really is under half a millisecond has a zero, and zero is a figure.
function msFigure(ms) {
  if (!Number.isFinite(ms) || ms < 0) return '—';
  return millis(Math.round(ms));
}

function rateFigure(rate) {
  if (!Number.isFinite(rate) || rate <= 0) return '—';
  return `${rate.toFixed(1)} tok/s`;
}

// ── misc ─────────────────────────────────────────────────
function escapeHtml(s) {
  const d = document.createElement('div');
  d.textContent = String(s ?? '');
  return d.innerHTML;
}

function alertErr(err) { alert(err.message || String(err)); }

connect();
