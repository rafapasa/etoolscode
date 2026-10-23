import { apiGet } from './api.js';
import { state } from './state.js';

let editor = null;
let monacoReady = false;
let currentPath = '';

function langFromPath(p) {
  if (!p) return 'plaintext';
  if (p.endsWith('.go')) return 'go';
  if (p.endsWith('.js')) return 'javascript';
  if (p.endsWith('.ts') || p.endsWith('.tsx')) return 'typescript';
  if (p.endsWith('.json')) return 'json';
  if (p.endsWith('.html') || p.endsWith('.htm')) return 'html';
  if (p.endsWith('.css')) return 'css';
  if (p.endsWith('.md')) return 'markdown';
  if (p.endsWith('.py')) return 'python';
  if (p.endsWith('.yaml') || p.endsWith('.yml')) return 'yaml';
  if (p.endsWith('.sh')) return 'shell';
  if (p.endsWith('.sql')) return 'sql';
  return 'plaintext';
}

function ensureEditor() {
  return new Promise((resolve) => {
    if (monacoReady && editor) { resolve(); return; }
    const el = document.getElementById('fonteEditor');
    if (!el) { resolve(); return; }
    if (typeof require === 'undefined') {
      el.textContent = 'Monaco CDN indisponível';
      resolve();
      return;
    }
    require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } });
    require(['vs/editor/editor.main'], function () {
      if (editor) { monacoReady = true; resolve(); return; }
      editor = monaco.editor.create(el, {
        value: '',
        language: 'go',
        theme: 'vs-dark',
        readOnly: true,
        automaticLayout: true,
        minimap: { enabled: true },
        fontSize: 13,
        wordWrap: 'off',
        scrollBeyondLastLine: false,
      });
      monacoReady = true;
      resolve();
    });
  });
}

export function initFontePanel() {
  // nothing extra; openSourceFile drives it
}

export async function openSourceFile(project, relPath) {
  if (!project || !relPath) return;
  currentPath = relPath;
  const pathLabel = document.getElementById('fontePath');
  if (pathLabel) pathLabel.textContent = project + ' / ' + relPath;

  // switch to fonte tab via app
  window.dispatchEvent(new CustomEvent('tab:fonte'));

  await ensureEditor();
  try {
    const data = await apiGet(
      '/api/source/' + encodeURIComponent(project) + '/' + relPath.split('/').map(encodeURIComponent).join('/')
    );
    const content = data.content || '';
    const lang = langFromPath(relPath);
    if (editor && window.monaco) {
      const model = editor.getModel();
      if (model) {
        monaco.editor.setModelLanguage(model, lang);
        editor.setValue(content);
      }
    }
  } catch (e) {
    if (editor) editor.setValue('// erro ao carregar: ' + (e.message || e));
  }
}
