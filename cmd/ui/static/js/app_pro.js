(function () {
  const $ = (id) => document.getElementById(id);
  let diffEditor = null;
  let originalModel = null;
  let modifiedModel = null;
  let currentProject = localStorage.getItem('etools_pro_project') || '';

  function log(msg) {
    const el = $('logPane');
    const t = new Date().toLocaleTimeString('pt-BR');
    el.textContent = `[${t}] ${msg}\n` + (el.textContent || '');
  }

  function project() {
    return ($('projectSelect').value || currentProject || '').trim();
  }

  async function api(path, opts) {
    const r = await fetch(path, opts);
    const text = await r.text();
    let data;
    try { data = JSON.parse(text); } catch { data = { raw: text, error: text }; }
    if (!r.ok) throw new Error(data.error || text || r.statusText);
    return data;
  }

  function langFromPath(p) {
    if (!p) return 'plaintext';
    if (p.endsWith('.go')) return 'go';
    if (p.endsWith('.js')) return 'javascript';
    if (p.endsWith('.ts')) return 'typescript';
    if (p.endsWith('.json')) return 'json';
    if (p.endsWith('.html')) return 'html';
    if (p.endsWith('.css')) return 'css';
    if (p.endsWith('.md')) return 'markdown';
    if (p.endsWith('.py')) return 'python';
    return 'plaintext';
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

  async function loadProjects() {
    try {
      const list = await api('/api/projects');
      const sel = $('projectSelect');
      sel.innerHTML = '<option value="">— selecione —</option>';
      (list || []).forEach(name => {
        const opt = document.createElement('option');
        opt.value = name;
        opt.textContent = name;
        if (name === currentProject) opt.selected = true;
        sel.appendChild(opt);
      });
      if (currentProject) await refreshStatus();
    } catch (e) {
      log('projetos: ' + e.message);
    }
  }

  async function refreshStatus() {
    const p = project();
    if (!p) {
      $('fileList').innerHTML = '<div class="status-pill">Selecione um projeto</div>';
      return;
    }
    localStorage.setItem('etools_pro_project', p);
    currentProject = p;
    try {
      const data = await api('/api/git/status?project=' + encodeURIComponent(p));
      $('branchInfo').textContent = (data.branch ? ('branch: ' + data.branch) : 'sem branch') + (data.root ? ' · ' + data.root : '');
      const files = data.files || [];
      if (!files.length) {
        $('fileList').innerHTML = '<div class="status-pill">Nenhuma alteração</div>';
      } else {
        $('fileList').innerHTML = files.map(f =>
          `<div class="file" data-path="${String(f.path).replace(/"/g, '&quot;')}" title="${f.index}${f.work} ${f.path}">${f.path}</div>`
        ).join('');
      }
      log('status: ' + files.length + ' arquivo(s) em ' + (data.root || p));
      await loadAudit(p);
    } catch (e) {
      log('status erro: ' + e.message);
      $('fileList').innerHTML = '<div class="status-pill">' + e.message + '</div>';
    }
  }

  async function showDiff(path) {
    const p = project();
    try {
      const data = await api('/api/git/file?project=' + encodeURIComponent(p) + '&file=' + encodeURIComponent(path));
      const lang = langFromPath(path);
      if (diffEditor && originalModel && modifiedModel) {
        monaco.editor.setModelLanguage(originalModel, lang);
        monaco.editor.setModelLanguage(modifiedModel, lang);
        originalModel.setValue(data.original || '');
        modifiedModel.setValue(data.modified || '');
      } else {
        $('diffEditor').textContent = (data.modified || data.original || '(vazio)');
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
    const p = project();
    if (!name) return alert('Informe o nome da branch (ex: dsac/' + p + '/minha-issue)');
    if (!p) return alert('Selecione um projeto');
    try {
      const data = await api('/api/git/branch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, project: p }),
      });
      log('branch: ' + data.branch);
      $('branchInfo').textContent = 'branch: ' + data.branch;
      await refreshStatus();
    } catch (e) {
      log('branch erro: ' + e.message);
      alert(e.message);
    }
  }

  async function loadAudit(p) {
    try {
      const data = await api('/api/audit/recent?limit=30&project=' + encodeURIComponent(p));
      const items = data.items || [];
      if (!items.length) {
        log('audit vazio para ' + p);
        return;
      }
      const lines = items.map(i => {
        const t = (i.timestamp || '').toString().replace('T', ' ').slice(0, 19);
        const mark = i.success === false ? 'FAIL' : 'ok';
        return `${t} [${mark}] ${i.tool || ''} ${i.path || i.issue_id || ''}`;
      });
      $('logPane').textContent = lines.join('\n');
    } catch (e) {
      log('audit: ' + e.message);
    }
  }

  $('btnRefresh').addEventListener('click', refreshStatus);
  $('btnBranch').addEventListener('click', createBranch);
  $('projectSelect').addEventListener('change', refreshStatus);
  $('fileList').addEventListener('click', (e) => {
    const el = e.target.closest('.file');
    if (el) showDiff(el.dataset.path);
  });

  initMonaco();
  loadProjects();
})();
