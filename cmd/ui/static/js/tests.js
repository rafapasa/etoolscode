import { apiGet, apiPost } from './api.js';
import { state } from './state.js';

export function initTestsPanel() {
  const btn = document.getElementById('btnRunTests');
  const sug = document.getElementById('btnSuggestTests');
  if (btn) btn.addEventListener('click', runTests);
  if (sug) sug.addEventListener('click', suggestTests);
}

async function suggestTests() {
  const project = state.currentProject;
  if (!project) { setOut('Selecione um projeto no combo superior.'); return; }
  try {
    const data = await apiGet('/api/test/suggest?project=' + encodeURIComponent(project));
    const input = document.getElementById('testCommand');
    if (input && data.command) input.value = data.command;
    setOut(`Sugerido (${data.reason}): ${data.command || '(nenhum)'}\nRoot: ${data.root || ''}`);
  } catch (e) {
    setOut('Erro: ' + (e.message || e));
  }
}

async function runTests() {
  const project = state.currentProject;
  if (!project) { setOut('Selecione um projeto no combo superior.'); return; }
  const command = (document.getElementById('testCommand')?.value || '').trim();
  setOut('▶ Executando...\n');
  try {
    const res = await apiPost('/api/test/run', { project, command });
    const status = res.ok ? '✅ PASSOU' : '❌ FALHOU';
    setOut(`${status}\n$ ${res.command}\n\n${res.output || res.error || ''}`);
  } catch (e) {
    setOut('Erro: ' + (e.message || e));
  }
}

function setOut(t) {
  const el = document.getElementById('testOutput');
  if (el) el.textContent = t;
}
