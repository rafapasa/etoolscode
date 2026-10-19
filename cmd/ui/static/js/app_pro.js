(function () {
  const $ = (id) => document.getElementById(id);
  let diffEditor = null;
  let originalModel = null;
  let modifiedModel = null;
  let currentRoot = localStorage.getItem('etools_pro_root') || '';

  function log(msg) {
    const el = $('logPane');
    const t = new Date().toLocaleTimeString('pt-BR');
    el.textContent = `[${t}] ${msg}\n` + el.textContent;
  }

  function root() {
    return ($('projectRoot').value || currentRoot || '').trim();
  }

  async function api(path, opts) {
    const r = await fetch(path, opts);
    const text = await r.text();
    let data;
    try { data = JSON.parse(text); } catch { data = { raw: text }; }
    if (!r.ok) throw new Error(data.error || text || r.statusText);
    return data;
  }

  function initMonaco() {
    if (typeof require === 'undefined') {
      log('Monaco CDN indisponível');
      return;
    }
    require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } });
    require(['vs/editor/editor.main'], function () {
      diffEditor = monaco.editor.createDiffEditor($('diffEditor'), {
        theme: 'vs-dark',
        readOnly: true,
        renderSideBySide: true,
        automaticLayout: true,
      });
      originalModel = monaco.editor.createModel('', 'go');
      modifiedModel = monaco.editor.createModel('', 'go');
      diffEditor.setModel({ original: originalModel, modified: modifiedModel });
      log('Monaco diff pronto');
    });
  }

  async function refreshStatus() {
    const r = root();
    if (!r) {
      $('fileList').innerHTML = '<div class="status-pill">Informe a raiz do projeto</div>';
      return;
    }
    localStorage.setItem('etools_pro_root', r);
    $('projectRoot').value = r;
    try {
      const data = await api('/api/git/status?root=' + encodeURIComponent(r));
      $('branchInfo').textContent = data.branch ? ('branch: ' + data.branch) : 'sem branch';
      const files = data.files || [];
      if (!files.length) {
        $('fileList').innerHTML = '<div class="status-pill">Nenhuma alteração</div>';
      } else {
        $('fileList').innerHTML = files.map(f =>
          `<div class="file" data-path="${f.path.replace(/"/g, '&quot;')}" title="${f.index}${f.work} ${f.path}">${f.path}</div>`
        ).join('');
      }
      log('status: ' + files.length + ' arquivo(s)');
      await loadAudit();
    } catch (e) {
      log('status erro: ' + e.message);
      $('fileList').innerHTML = '<div class="status-pill">' + e.message + '</div>';
    }
  }

  async function showDiff(path) {
    const r = root();
    try {
      const data = await api('/api/git/diff?root=' + encodeURIComponent(r) + '&file=' + encodeURIComponent(path));
      const diff = data.diff || '';
      // Monaco diff editor espera dois conteúdos; para unified diff simples, mostramos no modified.
      if (diffEditor && originalModel && modifiedModel) {
        originalModel.setValue('');
        modifiedModel.setValue(diff || '(sem diff)');
      } else {
        $('diffEditor').textContent = diff || '(sem diff)';
      }
      log('diff: ' + path);
      document.querySelectorAll('#fileList .file').forEach(el => {
        el.classList.toggle('active', el.dataset.path === path);
      });
    } catch (e) {
      log('diff erro: ' + e.message);
    }
  }

  async function createBranch() {
    const name = $('branchName').value.trim();
    const r = root();
    if (!name) return alert('Informe o nome da branch');
    if (!r) return alert('Informe a raiz');
    try {
      const data = await api('/api/git/branch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, root: r }),
      });
      log('branch criada: ' + data.branch);
      $('branchInfo').textContent = 'branch: ' + data.branch;
      await refreshStatus();
    } catch (e) {
      log('branch erro: ' + e.message);
      alert(e.message);
    }
  }

  async function loadAudit() {
    try {
      const data = await api('/api/audit/recent?limit=20');
      const items = data.items || [];
      if (!items.length) return;
      const lines = items.map(i => {
        const t = (i.timestamp || '').toString().slice(11, 19);
        return `${t} ${i.tool || ''} ${i.path || i.issue_id || ''} ${i.success === false ? 'FAIL' : 'ok'}`;
      });
      $('logPane').textContent = lines.join('\n') + '\n' + $('logPane').textContent;
    } catch (_) { /* audit opcional */ }
  }

  $('btnRefresh').addEventListener('click', refreshStatus);
  $('btnBranch').addEventListener('click', createBranch);
  $('fileList').addEventListener('click', (e) => {
    const el = e.target.closest('.file');
    if (el) showDiff(el.dataset.path);
  });
  if (currentRoot) $('projectRoot').value = currentRoot;
  initMonaco();
  refreshStatus();
})();
