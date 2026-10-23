import { apiGet, apiPost } from './api.js';
import { state } from './state.js';

let styles = [];
let currentPlan = null;

export async function initEngineering() {
  const btn = document.getElementById('btnEngineering');
  if (btn) btn.addEventListener('click', openEngineeringModal);
  const bPlan = document.getElementById('btnEngPlan');
  const bScaffold = document.getElementById('btnEngScaffold');
  const bDetect = document.getElementById('btnEngDetect');
  const bPreview = document.getElementById('btnEngPreview');
  if (bPlan) bPlan.addEventListener('click', buildPlan);
  if (bScaffold) bScaffold.addEventListener('click', runScaffold);
  if (bDetect) bDetect.addEventListener('click', detectStyle);
  if (bPreview) bPreview.addEventListener('click', previewScaffold);
  document.querySelectorAll('[data-close="engineeringModal"]').forEach(b => {
    b.addEventListener('click', closeEngineeringModal);
  });
  try {
    styles = await apiGet('/api/engineering/styles');
    const sel = document.getElementById('engStyle');
    if (sel && Array.isArray(styles)) {
      sel.innerHTML = styles.map(s =>
        `<option value="${s.id}" title="${escapeHtml(s.description)}">${escapeHtml(s.name)}</option>`
      ).join('');
    }
  } catch (e) {
    console.warn('[eng] styles', e);
  }
}

export function openEngineeringModal() {
  const m = document.getElementById('engineeringModal');
  if (!m) return;
  const proj = document.getElementById('engProject');
  if (proj) proj.value = state.currentProject || '';
  const alert = document.getElementById('engAlert');
  if (alert) { alert.textContent = ''; alert.className = 'alert'; }
  const preview = document.getElementById('engPlanPreview');
  if (preview) preview.textContent = 'Gere o plano para revisar aqui.';
  currentPlan = null;
  m.classList.add('open');
}

export function closeEngineeringModal() {
  const m = document.getElementById('engineeringModal');
  if (m) m.classList.remove('open');
}

async function detectStyle() {
  const project = val('engProject') || state.currentProject;
  if (!project) { show('Informe o projeto', false); return; }
  try {
    const data = await apiGet('/api/engineering/detect?project=' + encodeURIComponent(project));
    const sel = document.getElementById('engStyle');
    if (sel && data.style) sel.value = data.style;
    const pkg = document.getElementById('engPackage');
    if (pkg && !pkg.value && Array.isArray(data.internal) && data.internal.length) {
      // sugere service se existir
      const prefer = data.internal.find(p => p.includes('service')) || data.internal[0];
      pkg.placeholder = prefer;
    }
    show(`Estilo sugerido: ${data.style} (${data.reason})`, true);
    const preview = document.getElementById('engPlanPreview');
    if (preview) preview.textContent = JSON.stringify(data, null, 2);
  } catch (e) {
    show(e.message || String(e), false);
  }
}

async function buildPlan() {
  const body = {
    project: val('engProject'),
    style: val('engStyle'),
    feature: val('engFeature'),
    package_path: val('engPackage'),
    description: val('engDescription'),
    branch: val('engBranch'),
    enrich_llm: document.getElementById('engEnrichLLM')?.checked === true,
  };
  if (!body.feature || !body.description) {
    show('Preencha feature e descrição', false);
    return;
  }
  try {
    const res = await apiPost('/api/engineering/plan', body);
    currentPlan = res.plan || res;
    const preview = document.getElementById('engPlanPreview');
    if (preview) preview.textContent = JSON.stringify(currentPlan, null, 2) +
      (res.enrich_error ? '\n\n// enrich: ' + res.enrich_error : '');
    show(res.enrich_error ? 'Plano gerado (LLM parcial: ' + res.enrich_error + ')' : 'Plano gerado — revise e siga para preview/scaffold', true);
  } catch (e) {
    show(e.message || String(e), false);
  }
}

async function previewScaffold() {
  if (!currentPlan) { show('Gere o plano antes', false); return; }
  try {
    const res = await apiPost('/api/engineering/preview', { plan: currentPlan });
    const preview = document.getElementById('engPlanPreview');
    if (preview) {
      let text = '';
      const files = res.files || {};
      for (const [path, content] of Object.entries(files)) {
        text += '// ===== ' + path + ' =====\n' + content + '\n\n';
      }
      preview.textContent = text || JSON.stringify(res, null, 2);
    }
    show('Preview dos arquivos (não gravado)', true);
  } catch (e) {
    show(e.message || String(e), false);
  }
}

async function runScaffold() {
  if (!currentPlan) {
    show('Gere e revise o plano antes', false);
    return;
  }
  const project = val('engProject') || state.currentProject;
  if (!project) {
    show('Selecione um projeto', false);
    return;
  }
  try {
    const res = await apiPost('/api/engineering/scaffold', {
      project,
      plan: currentPlan,
      skip_existing: document.getElementById('engSkipExisting')?.checked !== false,
      create_issue: document.getElementById('engCreateIssue')?.checked === true,
      issue_name: val('engIssueName'),
    });
    const preview = document.getElementById('engPlanPreview');
    if (preview) preview.textContent = JSON.stringify(res, null, 2);
    show('Scaffold gravado no código do projeto' + (res.scaffold?.issue ? ' + issue criada' : ''), true);
    window.dispatchEvent(new CustomEvent('explorer:reload'));
  } catch (e) {
    show(e.message || String(e), false);
  }
}

function val(id) {
  return (document.getElementById(id)?.value || '').trim();
}
function show(msg, ok) {
  const alert = document.getElementById('engAlert');
  if (!alert) return;
  alert.textContent = msg;
  alert.className = 'alert ' + (ok ? 'ok' : 'err');
}
function escapeHtml(s) {
  return String(s ?? '').replace(/[&<>"']/g, c => ({
    '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'
  }[c]));
}
