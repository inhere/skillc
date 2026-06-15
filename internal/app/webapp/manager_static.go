package webapp

const managerHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Skillc Web Manager</title>
<style>
:root {
  color-scheme: light;
  --bg: #f6f7f4;
  --panel: #ffffff;
  --panel-2: #eef2ef;
  --line: #d8ded8;
  --line-strong: #b7c1bb;
  --text: #1c2620;
  --muted: #607069;
  --accent: #0f6b5f;
  --accent-soft: #dcefeb;
  --warn: #9a5b11;
  --warn-soft: #fff1d9;
  --bad: #a93535;
  --bad-soft: #f8e0dd;
  --ok: #28633d;
  --ok-soft: #dfeee4;
}
* { box-sizing: border-box; }
html, body { height: 100%; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 14px;
  letter-spacing: 0;
}
button, input, select, textarea { font: inherit; letter-spacing: 0; }
button {
  border: 1px solid var(--line-strong);
  background: var(--panel);
  color: var(--text);
  border-radius: 6px;
  min-height: 32px;
  padding: 0 10px;
  cursor: pointer;
}
button:hover { border-color: var(--accent); color: var(--accent); }
button.primary { background: var(--accent); border-color: var(--accent); color: white; }
button.primary:hover { color: white; background: #0b5b51; }
button:disabled { cursor: not-allowed; opacity: .45; }
button.danger { border-color: #c77b72; color: var(--bad); }
button.danger:hover { background: var(--bad-soft); }
input, select, textarea {
  min-height: 32px;
  border: 1px solid var(--line-strong);
  border-radius: 6px;
  background: var(--panel);
  color: var(--text);
  padding: 0 9px;
}
textarea {
  width: 100%;
  min-height: 96px;
  padding: 8px 9px;
  resize: vertical;
}
.shell { min-height: 100%; display: grid; grid-template-columns: 236px minmax(0, 1fr); }
.sidebar {
  background: #243029;
  color: #e8eee9;
  border-right: 1px solid #1c2721;
  display: flex;
  flex-direction: column;
}
.brand { padding: 18px 18px 14px; border-bottom: 1px solid rgba(255,255,255,.09); }
.brand h1 { margin: 0; font-size: 18px; line-height: 1.2; font-weight: 700; }
.brand p { margin: 5px 0 0; color: #b7c6bd; font-size: 12px; }
.nav { padding: 10px 8px; display: grid; gap: 3px; }
.nav button {
  width: 100%;
  justify-content: flex-start;
  text-align: left;
  border-color: transparent;
  background: transparent;
  color: #d9e3dc;
  padding: 0 10px;
}
.nav button.active, .nav button:hover { background: rgba(255,255,255,.09); color: white; border-color: rgba(255,255,255,.04); }
.sidebar-foot { margin-top: auto; padding: 12px 18px 16px; color: #aebdb4; font-size: 12px; border-top: 1px solid rgba(255,255,255,.09); }
.workspace { min-width: 0; display: flex; flex-direction: column; }
.topbar {
  min-height: 72px;
  padding: 14px 22px;
  border-bottom: 1px solid var(--line);
  background: rgba(255,255,255,.78);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}
.topbar h2 { margin: 0; font-size: 18px; line-height: 1.25; }
.path { margin-top: 5px; color: var(--muted); font-size: 12px; overflow-wrap: anywhere; }
.controls { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; justify-content: flex-end; }
.controls label { display: grid; gap: 4px; color: var(--muted); font-size: 11px; }
.actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.content { padding: 18px 22px 28px; overflow: auto; }
.view { display: none; }
.view.active { display: block; }
.metrics { display: grid; grid-template-columns: repeat(4, minmax(130px, 1fr)); gap: 10px; margin-bottom: 16px; }
.metric {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 12px;
  min-height: 78px;
}
.metric .label { color: var(--muted); font-size: 12px; }
.metric .value { margin-top: 8px; font-size: 24px; line-height: 1; font-weight: 700; }
.grid { display: grid; gap: 16px; }
.grid.two { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
.section { margin-bottom: 18px; }
.section-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 8px; }
.section h3 { margin: 0; font-size: 15px; }
.section .hint { color: var(--muted); font-size: 12px; }
.panel {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 12px;
}
.table-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; background: var(--panel); }
table { width: 100%; border-collapse: collapse; min-width: 680px; }
th, td { padding: 9px 10px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
th { background: var(--panel-2); color: var(--muted); font-size: 12px; font-weight: 650; }
tbody tr:last-child td { border-bottom: 0; }
td.wrap { overflow-wrap: anywhere; }
.status { display: inline-flex; align-items: center; min-height: 22px; padding: 0 7px; border-radius: 999px; font-size: 12px; border: 1px solid var(--line); background: var(--panel-2); color: var(--muted); white-space: nowrap; }
.status.installed { background: var(--ok-soft); color: var(--ok); border-color: #bad8c4; }
.status.outdated, .status.missing { background: var(--warn-soft); color: var(--warn); border-color: #efd2a4; }
.status.orphan, .status.unmanaged, .status.source-error { background: var(--bad-soft); color: var(--bad); border-color: #edb9b2; }
.toolbar-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 10px; }
.toolbar-row input { width: min(340px, 100%); }
.inline-check { display: inline-flex; align-items: center; gap: 6px; color: var(--muted); }
.inline-check input { min-height: auto; width: auto; }
.empty { padding: 18px; color: var(--muted); border: 1px dashed var(--line-strong); border-radius: 8px; background: rgba(255,255,255,.5); }
.error { padding: 10px 12px; background: var(--bad-soft); color: var(--bad); border: 1px solid #edb9b2; border-radius: 8px; margin-bottom: 12px; }
.plan {
  white-space: pre-wrap;
  word-break: break-word;
  background: #202923;
  color: #edf4ef;
  border-radius: 8px;
  padding: 12px;
  min-height: 80px;
  overflow: auto;
}
.stack { display: grid; gap: 12px; }
.muted { color: var(--muted); }
.mono { font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace; font-size: 12px; }
@media (max-width: 920px) {
  .shell { grid-template-columns: 1fr; }
  .sidebar { position: static; }
  .nav { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .topbar { align-items: flex-start; flex-direction: column; }
  .controls { justify-content: flex-start; }
  .metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .grid.two { grid-template-columns: 1fr; }
}
@media (max-width: 560px) {
  .nav { grid-template-columns: 1fr 1fr; }
  .metrics { grid-template-columns: 1fr; }
  .content { padding: 14px; }
}
</style>
</head>
<body>
<div class="shell">
  <aside class="sidebar">
    <div class="brand">
      <h1>Skillc</h1>
      <p>Local Web Manager</p>
    </div>
    <nav class="nav" aria-label="Main">
      <button class="active" data-view="dashboard">Dashboard</button>
      <button data-view="sources">Sources</button>
      <button data-view="profiles">Profiles</button>
      <button data-view="skills">Skills</button>
      <button data-view="projects">Projects</button>
      <button data-view="drift">Version Drift</button>
    </nav>
    <div class="sidebar-foot">Plan-first management with guarded current-project execution.</div>
  </aside>
  <main class="workspace">
    <header class="topbar">
      <div>
        <h2 id="page-title">Dashboard</h2>
        <div class="path" id="project-path">Loading project...</div>
      </div>
      <div class="controls">
        <label>Agent <input id="agent-input" value="universal" autocomplete="off"></label>
        <label>Scope
          <select id="scope-input">
            <option value="project">project</option>
            <option value="user">user</option>
          </select>
        </label>
        <button class="primary" id="refresh-btn">Refresh</button>
      </div>
    </header>
    <div class="content">
      <div id="errors"></div>
      <section id="view-dashboard" class="view active">
        <div class="metrics" id="metrics"></div>
        <div class="grid two">
          <section class="section panel">
            <div class="section-head"><h3>Status</h3><span class="hint">Current project</span></div>
            <div id="status-summary"></div>
          </section>
          <section class="section panel">
            <div class="section-head"><h3>Update Candidates</h3><button id="plan-update-btn">Plan update</button></div>
            <div id="update-candidates"></div>
          </section>
        </div>
      </section>

      <section id="view-sources" class="view">
        <div class="section-head"><h3>Sources</h3><span class="hint" id="sources-count"></span></div>
        <div class="panel section">
          <div class="toolbar-row">
            <input id="source-value-input" placeholder="Local path or git URL" autocomplete="off">
            <input id="source-ref-input" placeholder="Git ref" autocomplete="off">
            <label class="inline-check"><input id="source-sync-input" type="checkbox"> sync</label>
            <button id="plan-source-add-btn">Plan add</button>
          </div>
        </div>
        <div id="sources-table"></div>
      </section>

      <section id="view-profiles" class="view">
        <div class="section-head"><h3>Profiles</h3><span class="hint">Plan first, then confirm apply</span></div>
        <div class="panel section">
          <div class="stack">
            <div class="toolbar-row">
              <input id="profile-name-input" placeholder="Name" autocomplete="off">
              <input id="profile-description-input" placeholder="Description" autocomplete="off">
              <input id="profile-agent-input" placeholder="Default agent" autocomplete="off">
              <select id="profile-scope-input">
                <option value="project">project</option>
                <option value="user">user</option>
              </select>
              <select id="profile-install-mode-input">
                <option value="">install mode</option>
                <option value="copy">copy</option>
                <option value="symlink">symlink</option>
              </select>
            </div>
            <textarea id="profile-targets-input" placeholder="gstack go-pro"></textarea>
            <div class="toolbar-row">
              <button id="plan-profile-save-btn">Plan save</button>
              <button id="plan-profile-installed-btn">From installed</button>
              <input id="profile-collection-input" placeholder="source/collection" autocomplete="off">
              <button id="plan-profile-collection-btn">From collection</button>
            </div>
          </div>
        </div>
        <div id="profiles-table"></div>
      </section>

      <section id="view-skills" class="view">
        <div class="toolbar-row">
          <input id="skill-search" placeholder="Search indexed skills" autocomplete="off">
          <button id="skill-search-btn">Search</button>
        </div>
        <div id="skills-table"></div>
      </section>

      <section id="view-projects" class="view">
        <div class="section-head"><h3>Projects / Install Map</h3><span class="hint">Derived from lock records</span></div>
        <div id="install-map-table"></div>
      </section>

      <section id="view-drift" class="view">
        <div class="section-head"><h3>Version Drift</h3><span class="hint">Grouped by source-qualified identity</span></div>
        <div id="drift-table"></div>
      </section>

      <section class="section">
        <div class="section-head">
          <h3>Plan Output</h3>
          <div class="actions" id="action-bar">
            <button class="danger" id="apply-profile-btn" disabled>Apply profile</button>
            <button class="danger" id="run-update-btn" disabled>Run update</button>
            <button class="danger" id="run-source-action-btn" disabled>Run source action</button>
            <button class="danger" id="run-profile-action-btn" disabled>Run profile action</button>
            <button class="danger" id="run-uninstall-btn" disabled>Run uninstall</button>
          </div>
        </div>
        <pre id="plan-output" class="plan">No plan requested.</pre>
      </section>
    </div>
  </main>
</div>
<script>
(function () {
  var state = {
    summary: null,
    sources: [],
    profiles: [],
    status: { items: [], summary: {} },
    installs: [],
    drift: [],
    skills: [],
    pendingAction: null
  };

  function byId(id) { return document.getElementById(id); }
  function esc(value) {
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }
  function reqParams() {
    var params = new URLSearchParams();
    var agent = byId('agent-input').value.trim();
    var scope = byId('scope-input').value;
    if (agent) params.set('agent', agent);
    if (scope) params.set('scope', scope);
    return params.toString();
  }
  function api(path, options) {
    var suffix = reqParams();
    var sep = path.indexOf('?') >= 0 ? '&' : '?';
    var url = suffix ? path + sep + suffix : path;
    return fetch(url, options || {}).then(function (resp) {
      return resp.json().then(function (data) {
        if (!resp.ok) throw new Error(data.error || resp.statusText);
        return data;
      });
    });
  }
  function postJSON(path, payload) {
    return api(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload || {})
    });
  }
  function setPendingAction(action) {
    state.pendingAction = action;
    byId('apply-profile-btn').disabled = !(action && action.type === 'profile');
    byId('run-update-btn').disabled = !(action && action.type === 'update');
    byId('run-source-action-btn').disabled = !(action && action.type.indexOf('source-') === 0);
    byId('run-profile-action-btn').disabled = !(action && action.type.indexOf('profile-') === 0);
    byId('run-uninstall-btn').disabled = !(action && action.type === 'uninstall');
  }
  function showError(err) {
    byId('errors').innerHTML = '<div class="error">' + esc(err.message || err) + '</div>';
  }
  function clearError() { byId('errors').innerHTML = ''; }
  function table(headers, rows, emptyText) {
    if (!rows || rows.length === 0) return '<div class="empty">' + esc(emptyText || 'No data') + '</div>';
    return '<div class="table-wrap"><table><thead><tr>' +
      headers.map(function (h) { return '<th>' + esc(h) + '</th>'; }).join('') +
      '</tr></thead><tbody>' + rows.join('') + '</tbody></table></div>';
  }
  function statusPill(value) {
    var cls = String(value || '').replace(/[^a-z0-9-]/g, '');
    return '<span class="status ' + esc(cls) + '">' + esc(value || 'unknown') + '</span>';
  }
  function renderMetrics() {
    var summary = state.summary || {};
    var status = summary.status || {};
    var items = [
      ['Sources', summary.source_count || 0],
      ['Profiles', summary.profile_count || 0],
      ['Indexed Skills', summary.skill_count || 0],
      ['Outdated', status.outdated || 0],
      ['Missing', status.missing || 0],
      ['Installed', status.installed || 0],
      ['Unmanaged', status.unmanaged || 0],
      ['Source Errors', status.source_error || 0]
    ];
    byId('metrics').innerHTML = items.map(function (item) {
      return '<div class="metric"><div class="label">' + esc(item[0]) + '</div><div class="value">' + esc(item[1]) + '</div></div>';
    }).join('');
    byId('project-path').textContent = summary.project_path || 'Project path unavailable';
  }
  function renderStatus() {
    var rows = (state.status.items || []).slice(0, 8).map(function (item) {
      return '<tr><td>' + statusPill(item.Status || item.status) + '</td><td class="wrap">' +
        esc(item.SourceQualifiedName || item.source_qualified_name || item.SkillID || item.skill_id) +
        '</td><td>' + esc(item.CurrentVersion || item.current_version || '') +
        '</td><td>' + esc(item.LatestVersion || item.latest_version || '') + '</td></tr>';
    });
    byId('status-summary').innerHTML = table(['Status', 'Skill', 'Current', 'Latest'], rows, 'No current project status items.');
    var candidates = (state.status.items || []).filter(function (item) {
      var status = item.Status || item.status;
      return status === 'outdated' || status === 'missing';
    });
    byId('update-candidates').innerHTML = candidates.length ?
      '<div class="muted">' + candidates.length + ' candidate(s) can be previewed.</div>' :
      '<div class="empty">No missing or outdated items.</div>';
  }
  function renderSources() {
    byId('sources-count').textContent = state.sources.length + ' source(s)';
    var rows = state.sources.map(function (s) {
      var id = s.ID || s.id;
      return '<tr><td>' + esc(s.ID || s.id) + '</td><td>' + esc(s.Name || s.name) +
        '</td><td>' + esc(s.Type || s.type) + '</td><td>' + statusPill(s.Status || s.status || 'configured') +
        '</td><td class="wrap mono">' + esc(s.Path || s.path || s.URL || s.url) +
        '</td><td>' + esc(s.Ref || s.ref || '') + '</td><td class="wrap">' + esc(s.ErrorMessage || s.error_message || '') +
        '</td><td><button data-source-sync="' + esc(id) + '">Plan sync</button> ' +
        '<button class="danger" data-source-remove="' + esc(id) + '">Plan remove</button></td></tr>';
    });
    byId('sources-table').innerHTML = table(['ID', 'Name', 'Type', 'Status', 'Path / URL', 'Ref', 'Error', ''], rows, 'No sources configured.');
    byId('sources-table').querySelectorAll('button[data-source-sync]').forEach(function (btn) {
      btn.addEventListener('click', function () { planSourceSync(btn.getAttribute('data-source-sync')); });
    });
    byId('sources-table').querySelectorAll('button[data-source-remove]').forEach(function (btn) {
      btn.addEventListener('click', function () { planSourceRemove(btn.getAttribute('data-source-remove')); });
    });
  }
  function renderProfiles() {
    var rows = state.profiles.map(function (p) {
      var targets = p.targets || p.Targets || [];
      var name = p.name || p.Name;
      return '<tr><td>' + esc(name) + '</td><td class="wrap">' + esc(p.description || p.Description || '') +
        '</td><td>' + esc(p.default_agent || p.DefaultAgent || '') +
        '</td><td>' + esc(p.default_scope || p.DefaultScope || '') +
        '</td><td>' + targets.length + '</td><td><button data-profile="' + esc(name) + '">Plan</button></td></tr>';
    });
    byId('profiles-table').innerHTML = table(['Name', 'Description', 'Agent', 'Scope', 'Targets', ''], rows, 'No profiles configured.');
    byId('profiles-table').querySelectorAll('button[data-profile]').forEach(function (btn) {
      btn.addEventListener('click', function () { planProfile(btn.getAttribute('data-profile')); });
    });
  }
  function renderSkills() {
    var rows = state.skills.map(function (s) {
      return '<tr><td>' + esc(s.source_qualified_name || s.SourceQualifiedName || s.id || s.ID) +
        '</td><td>' + esc(s.source_id || s.SourceID || '') + '</td><td>' + esc(s.collection || s.Collection || '') +
        '</td><td>' + esc(s.version || s.Version || '') + '</td><td class="wrap">' + esc(s.description || s.Description || '') + '</td></tr>';
    });
    byId('skills-table').innerHTML = table(['Skill', 'Source', 'Collection', 'Version', 'Description'], rows, 'Search or refresh to load indexed skills.');
  }
  function renderInstalls() {
    var rows = state.installs.map(function (item, idx) {
      var target = item.source_qualified_name || item.qualified_name || item.skill_id;
      return '<tr><td class="wrap mono">' + esc(item.project_path) + '</td><td>' + esc(item.scope) +
        '</td><td>' + esc(item.agent) + '</td><td>' + esc(item.profile || '') +
        '</td><td>' + esc(target) +
        '</td><td>' + esc(item.version || '') +
        '</td><td><button class="danger" data-uninstall-idx="' + idx + '">Plan uninstall</button></td></tr>';
    });
    byId('install-map-table').innerHTML = table(['Project', 'Scope', 'Agent', 'Profile', 'Skill', 'Version', ''], rows, 'No agent-attributed install records found.');
    byId('install-map-table').querySelectorAll('button[data-uninstall-idx]').forEach(function (btn) {
      btn.addEventListener('click', function () { planUninstall(Number(btn.getAttribute('data-uninstall-idx'))); });
    });
  }
  function renderDrift() {
    var rows = state.drift.map(function (group, idx) {
      var versions = (group.versions || []).map(function (bucket) {
        return esc(bucket.version || '(empty)') + ' (' + (bucket.projects || []).length + ')';
      }).join(', ');
      return '<tr><td>' + esc(group.source_qualified_name || group.skill_id) +
        '</td><td>' + esc(group.source_id || '') + '</td><td>' + esc(group.latest_version || '') +
        '</td><td class="wrap">' + versions + '</td><td><button data-drift="' + idx + '">Plan update</button></td></tr>';
    });
    byId('drift-table').innerHTML = table(['Skill', 'Source', 'Latest', 'Installed versions', ''], rows, 'No version drift found.');
    byId('drift-table').querySelectorAll('button[data-drift]').forEach(function (btn) {
      btn.addEventListener('click', planUpdate);
    });
  }
  function renderAll() {
    renderMetrics();
    renderStatus();
    renderSources();
    renderProfiles();
    renderSkills();
    renderInstalls();
    renderDrift();
  }
  function loadAll() {
    clearError();
    Promise.all([
      api('/api/summary'),
      api('/api/sources'),
      api('/api/profiles'),
      api('/api/status'),
      api('/api/install-map'),
      api('/api/version-drift'),
      api('/api/skills')
    ]).then(function (all) {
      state.summary = all[0];
      state.sources = all[1] || [];
      state.profiles = all[2] || [];
      state.status = all[3] || { items: [], summary: {} };
      state.installs = all[4] || [];
      state.drift = all[5] || [];
      state.skills = all[6] || [];
      renderAll();
    }).catch(showError);
  }
  function searchSkills() {
    clearError();
    var keyword = byId('skill-search').value.trim();
    var path = '/api/skills';
    if (keyword) path += '?keyword=' + encodeURIComponent(keyword);
    api(path).then(function (items) {
      state.skills = items || [];
      renderSkills();
    }).catch(showError);
  }
  function planProfile(name) {
    clearError();
    postJSON('/api/profiles/' + encodeURIComponent(name) + '/plan', {})
      .then(function (plan) {
        setPendingAction({ type: 'profile', name: name });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planUpdate() {
    clearError();
    postJSON('/api/update/plan', {})
      .then(function (plan) {
        setPendingAction({ type: 'update' });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planSourceAdd() {
    clearError();
    var payload = {
      value: byId('source-value-input').value.trim(),
      ref: byId('source-ref-input').value.trim(),
      sync: byId('source-sync-input').checked
    };
    postJSON('/api/sources/add/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'source-add', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planSourceSync(id) {
    clearError();
    var payload = { id: id };
    postJSON('/api/sources/sync/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'source-sync', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planSourceRemove(id) {
    clearError();
    var payload = { id: id };
    postJSON('/api/sources/remove/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'source-remove', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function profileTargetsFromText() {
    return byId('profile-targets-input').value.split(/\r?\n/).map(function (line) {
      return line.trim();
    }).filter(Boolean).map(function (line) {
      var parts = line.split(/\s+/);
      if (parts.length === 1) return { skill: parts[0] };
      return { source: parts[0], skill: parts.slice(1).join(' ') };
    });
  }
  function profileSavePayload() {
    return {
      name: byId('profile-name-input').value.trim(),
      description: byId('profile-description-input').value.trim(),
      default_agent: byId('profile-agent-input').value.trim(),
      default_scope: byId('profile-scope-input').value,
      install_mode: byId('profile-install-mode-input').value,
      targets: profileTargetsFromText()
    };
  }
  function planProfileSave() {
    clearError();
    var payload = profileSavePayload();
    postJSON('/api/profiles/save/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'profile-save', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planProfileFromInstalled() {
    clearError();
    var payload = {
      name: byId('profile-name-input').value.trim(),
      agent: byId('agent-input').value.trim(),
      scope: byId('scope-input').value
    };
    postJSON('/api/profiles/from-installed/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'profile-installed', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planProfileFromCollection() {
    clearError();
    var payload = {
      name: byId('profile-name-input').value.trim(),
      selector: byId('profile-collection-input').value.trim()
    };
    postJSON('/api/profiles/from-collection/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'profile-collection', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function planUninstall(idx) {
    clearError();
    var item = state.installs[idx];
    if (!item) return;
    var target = item.source_qualified_name || item.qualified_name || item.skill_id;
    var payload = {
      skills: [target],
      agent: item.agent || byId('agent-input').value.trim(),
      scope: item.scope || byId('scope-input').value
    };
    postJSON('/api/uninstall/plan', payload)
      .then(function (plan) {
        setPendingAction({ type: 'uninstall', payload: payload });
        byId('plan-output').textContent = JSON.stringify(plan, null, 2);
      })
      .catch(showError);
  }
  function applyProfile() {
    if (!state.pendingAction || state.pendingAction.type !== 'profile') return;
    if (!window.confirm('Apply this profile to the current project?')) return;
    postJSON('/api/profiles/' + encodeURIComponent(state.pendingAction.name) + '/apply', { confirm: true })
      .then(function (result) {
        byId('plan-output').textContent = JSON.stringify(result, null, 2);
        setPendingAction(null);
        loadAll();
      })
      .catch(showError);
  }
  function runUninstall() {
    var action = state.pendingAction;
    if (!action || action.type !== 'uninstall') return;
    if (!window.confirm('Run uninstall for the selected skill?')) return;
    var payload = Object.assign({ confirm: true }, action.payload || {});
    postJSON('/api/uninstall/run', payload)
      .then(function (result) {
        byId('plan-output').textContent = JSON.stringify(result, null, 2);
        setPendingAction(null);
        loadAll();
      })
      .catch(showError);
  }
  function runProfileAction() {
    var action = state.pendingAction;
    if (!action || action.type.indexOf('profile-') !== 0) return;
    if (!window.confirm('Run this profile action?')) return;
    var route = {
      'profile-save': '/api/profiles/save/run',
      'profile-installed': '/api/profiles/from-installed/run',
      'profile-collection': '/api/profiles/from-collection/run'
    }[action.type];
    var payload = Object.assign({ confirm: true }, action.payload || {});
    postJSON(route, payload)
      .then(function (result) {
        byId('plan-output').textContent = JSON.stringify(result, null, 2);
        setPendingAction(null);
        loadAll();
      })
      .catch(showError);
  }
  function runUpdate() {
    if (!state.pendingAction || state.pendingAction.type !== 'update') return;
    if (!window.confirm('Run update for the current project?')) return;
    postJSON('/api/update/run', { confirm: true })
      .then(function (result) {
        byId('plan-output').textContent = JSON.stringify(result, null, 2);
        setPendingAction(null);
        loadAll();
      })
      .catch(showError);
  }
  function runSourceAction() {
    var action = state.pendingAction;
    if (!action || action.type.indexOf('source-') !== 0) return;
    if (!window.confirm('Run this source action for the current project?')) return;
    var route = {
      'source-add': '/api/sources/add/run',
      'source-sync': '/api/sources/sync/run',
      'source-remove': '/api/sources/remove/run'
    }[action.type];
    var payload = Object.assign({ confirm: true }, action.payload || {});
    postJSON(route, payload)
      .then(function (result) {
        byId('plan-output').textContent = JSON.stringify(result, null, 2);
        setPendingAction(null);
        loadAll();
      })
      .catch(showError);
  }
  document.querySelectorAll('.nav button').forEach(function (btn) {
    btn.addEventListener('click', function () {
      document.querySelectorAll('.nav button').forEach(function (b) { b.classList.remove('active'); });
      document.querySelectorAll('.view').forEach(function (v) { v.classList.remove('active'); });
      btn.classList.add('active');
      byId('view-' + btn.getAttribute('data-view')).classList.add('active');
      byId('page-title').textContent = btn.textContent;
    });
  });
  byId('refresh-btn').addEventListener('click', loadAll);
  byId('skill-search-btn').addEventListener('click', searchSkills);
  byId('skill-search').addEventListener('keydown', function (event) {
    if (event.key === 'Enter') searchSkills();
  });
  byId('plan-update-btn').addEventListener('click', planUpdate);
  byId('plan-source-add-btn').addEventListener('click', planSourceAdd);
  byId('plan-profile-save-btn').addEventListener('click', planProfileSave);
  byId('plan-profile-installed-btn').addEventListener('click', planProfileFromInstalled);
  byId('plan-profile-collection-btn').addEventListener('click', planProfileFromCollection);
  byId('apply-profile-btn').addEventListener('click', applyProfile);
  byId('run-update-btn').addEventListener('click', runUpdate);
  byId('run-source-action-btn').addEventListener('click', runSourceAction);
  byId('run-profile-action-btn').addEventListener('click', runProfileAction);
  byId('run-uninstall-btn').addEventListener('click', runUninstall);
  setPendingAction(null);
  loadAll();
})();
</script>
</body>
</html>`
