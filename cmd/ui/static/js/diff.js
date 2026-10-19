import { apiGet } from './api.js';
import { state } from './state.js';

let diffEditor = null;
let originalModel = null;
let modifiedModel = null;
let monacoReady = false;
let activePath = '';

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
  if (p.endsWith('.dart')) return 'dart';
  return 'plaintext';
}

export function initDiffPanel() {
  const tabs = document.querySelectorAll('[data-files-tab]');
  tabs.forEach(btn => {
    btn.addEventListener('click', () => switchFilesTab(btn.dataset.filesTab));
  });
  const refresh = document.getElementById('btnDiffRefresh');
  if (refresh) refresh.addEventListener('click', () => loadDiffStatus());
  const list = document.getElementById('diffFileList');
  if (list) {
    list.addEventListener('click', (e) => {
      const row = e.target.closest('[data-diff-path]');
      if (row) showFileDiff(row.dataset.diffPath);
    });
  }
}

function switchFilesTab(tab) {
  document.querySelectorAll('[data-files-tab]').forEach(b => {
    b.classList.toggle('active', b.dataset.filesTab === tab);
  });
  const fontes = document.getElementById('filesPaneFontes');
  const diff = document.getElementById('filesPaneDiff');
  if (fontes) fontes.style.display = tab === 'fontes' ? 'flex' : 'none';
  if (diff) diff.style.display = tab === 'diff' ? 'flex' : 'none';
  if (tab === 'diff') {
    ensureMonaco().then(() => loadDiffStatus());
  }
}

function ensureMonaco() {
  return new Promise((resolve) => {
    if (monacoReady && diffEditor) {
      resolve();
      return;
    }
    const el = document.getElementById('diffEditor');
    if (!el) {
      resolve();
      return;
    }
    if (typeof require === 'undefined') {
      el.textContent = 'Monaco CDN indisponível';
      resolve();
      return;
    }
    require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } });
    require(['vs/editor/editor.main'], function () {
      if (diffEditor) {
        monacoReady = true;
        resolve();
        return;
      }
      diffEditor = monaco.editor.createDiffEditor(el, {
        theme: 'vs-dark',
        readOnly: true,
        renderSideBySide: true,
        automaticLayout: true,
        minimap: { enabled: false },
      });
      originalModel = monaco.editor.createModel('', 'go');
      modifiedModel = monaco.editor.createModel('', 'go');
      diffEditor.setModel({ original: originalModel, modified: modifiedModel });
      monacoReady = true;
      resolve();
    });
  });
}

export async function loadDiffStatus() {
  const list = document.getElementById('diffFileList');
  const info = document.getElementById('diffBranchInfo');
  if (!list) return;
  const project = state.currentProject;
  if (!project) {
    list.innerHTML = '<div class="explorer-empty">Selecione um projeto</div>';
    if (info) info.textContent = '—';
    return;
  }
  try {
    const data = await apiGet('/api/git/status?project=' + encodeURIComponent(project));
    if (info) {
      info.textContent = (data.branch ? data.branch : 'sem branch') + (data.root ? ' · ' + data.root : '');
      info.title = data.root || '';
    }
    const files = data.files || [];
    if (!files.length) {
      list.innerHTML = '<div class="explorer-empty">Nenhuma alteração</div>';
      if (originalModel) originalModel.setValue('');
      if (modifiedModel) modifiedModel.setValue('');
      return;
    }
    list.innerHTML = files.map(f => {
      const active = f.path === activePath ? ' active' : '';
      return `<div class="diff-file-row${active}" data-diff-path="${escapeAttr(f.path)}" title="${escapeAttr(f.path)}">
        <span class="diff-flag">${escapeHtml(f.index || ' ')}${escapeHtml(f.work || ' ')}</span>
        <span class="diff-name">${escapeHtml(f.path)}</span>
      </div>`;
    }).join('');
    if (activePath && files.some(f => f.path === activePath)) {
      await showFileDiff(activePath);
    } else if (files[0]) {
      await showFileDiff(files[0].path);
    }
  } catch (e) {
    list.innerHTML = `<div class="explorer-empty">${escapeHtml(e.message || 'Erro git')}</div>`;
    if (info) info.textContent = e.message || 'erro';
  }
}

async function showFileDiff(path) {
  const project = state.currentProject;
  if (!project || !path) return;
  activePath = path;
  document.querySelectorAll('#diffFileList .diff-file-row').forEach(el => {
    el.classList.toggle('active', el.dataset.diffPath === path);
  });
  await ensureMonaco();
  try {
    const data = await apiGet(
      '/api/git/file?project=' + encodeURIComponent(project) + '&file=' + encodeURIComponent(path)
    );
    const lang = langFromPath(path);
    if (originalModel && modifiedModel && window.monaco) {
      monaco.editor.setModelLanguage(originalModel, lang);
      monaco.editor.setModelLanguage(modifiedModel, lang);
      originalModel.setValue(data.original || '');
      modifiedModel.setValue(data.modified || '');
    }
  } catch (e) {
    if (modifiedModel) modifiedModel.setValue('// erro: ' + (e.message || e));
  }
}

function escapeHtml(s) {
  return String(s ?? '').replace(/[&<>"']/g, c => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]));
}
function escapeAttr(s) {
  return String(s ?? '').replace(/"/g, '&quot;');
}
