/** Painel de log de atividades (chat + issues) — rodapé retrátil */
const MAX_LINES = 500;
let open = localStorage.getItem('etools_actlog_open') !== '0';

function el(id){ return document.getElementById(id); }

export function initActivityLog(){
  const panel = el('activityLog');
  const toggle = el('btnActivityLogToggle');
  const clear = el('btnActivityLogClear');
  if (!panel) return;
  applyOpenState();
  if (toggle) toggle.addEventListener('click', () => {
    open = !open;
    localStorage.setItem('etools_actlog_open', open ? '1' : '0');
    applyOpenState();
  });
  if (clear) clear.addEventListener('click', () => {
    const body = el('activityLogBody');
    if (body) body.innerHTML = '';
  });
}

function applyOpenState(){
  const panel = el('activityLog');
  const toggle = el('btnActivityLogToggle');
  if (!panel) return;
  panel.classList.toggle('is-open', open);
  panel.classList.toggle('is-collapsed', !open);
  if (toggle) toggle.textContent = open ? '▾ Log' : '▸ Log';
  document.body.classList.toggle('activity-log-open', open);
}

function iconFor(kind, ok){
  if (!ok) return '⚠';
  switch (kind) {
    case 'write': return '✎';
    case 'list': return '☰';
    case 'command': return '▷';
    case 'git': return '⎇';
    case 'issue': return '▶';
    default: return '◎'; // read
  }
}

function verbFor(kind, ok, name){
  if (!ok) return 'Falhou';
  if (name === 'search_replace') return 'Patch';
  switch (kind) {
    case 'write': return 'Gravou';
    case 'list': return 'Listou';
    case 'command': return 'Cmd';
    case 'git': return 'Git';
    case 'issue': return 'Issue';
    default: return 'Leu';
  }
}

/**
 * @param {{kind?:string,name?:string,path?:string,detail?:string,error?:string,ok?:boolean,source?:string}} data
 */
export function pushActivity(data){
  const body = el('activityLogBody');
  if (!body) return;
  const kind = data.kind || 'read';
  const ok = data.ok !== false;
  const name = data.name || '';
  const path = (data.path || '').trim();
  const detail = (data.detail || data.error || '').trim();
  const source = data.source || '';
  const time = new Date().toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

  let main = path;
  if (!main && detail) main = detail.split('\n')[0];
  if (!main) main = name || '(sem path)';

  const row = document.createElement('div');
  row.className = 'act-row act-' + kind + (ok ? '' : ' act-err');
  row.title = [name, path, detail].filter(Boolean).join(' · ');
  row.innerHTML =
    `<span class="act-time">${time}</span>` +
    `<span class="act-icon">${iconFor(kind, ok)}</span>` +
    `<span class="act-verb">${verbFor(kind, ok, name)}</span>` +
    `<span class="act-main">${escapeHtml(main)}</span>` +
    (detail && detail !== main ? `<span class="act-detail">${escapeHtml(detail.length > 180 ? detail.slice(0,180)+'…' : detail)}</span>` : '') +
    (source ? `<span class="act-src">${escapeHtml(source)}</span>` : '');

  body.appendChild(row);
  while (body.children.length > MAX_LINES) body.removeChild(body.firstChild);
  body.scrollTop = body.scrollHeight;

  // se estiver colapsado e for erro, pisca o toggle
  if (!open && !ok) {
    const toggle = el('btnActivityLogToggle');
    if (toggle) {
      toggle.classList.add('has-error');
      setTimeout(() => toggle.classList.remove('has-error'), 2500);
    }
  }
}

export function pushIssueLine(text){
  const t = String(text || '');
  let kind = 'issue';
  let ok = true;
  if (/erro|falh|❌|AVISO: erro/i.test(t)) { ok = false; kind = 'issue'; }
  else if (/\[Tool\].*write|search_replace|Gravou/i.test(t)) kind = 'write';
  else if (/\[Tool\].*read|Leu/i.test(t)) kind = 'read';
  else if (/\[Tool\].*list/i.test(t)) kind = 'list';
  else if (/run_command|go test|go build/i.test(t)) kind = 'command';
  pushActivity({ kind, ok, path: t, detail: '', source: 'issue', name: 'issue' });
}

function escapeHtml(s){
  return String(s ?? '').replace(/[&<>"']/g, c => ({
    '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'
  }[c]));
}


/** Atualiza contador de turnos do chat (barra + valor). */
export function setChatTurnCounter(current, max, limitReached) {
  const root = document.getElementById('chatTurnCounter');
  const val = document.getElementById('chatTurnValue');
  const fill = document.getElementById('chatTurnFill');
  const btn = document.getElementById('btnChatContinueFooter');
  if (!root) return;
  if (!max || max < 1) {
    if (val) val.textContent = '—';
    if (fill) fill.style.width = '0%';
    root.classList.remove('is-warn', 'is-limit', 'is-active');
    root.title = 'Turnos LLM deste pedido no chat';
    if (btn) {
      btn.disabled = true;
      btn.classList.remove('is-pulse');
    }
    return;
  }
  const c = Math.max(0, current || 0);
  const pct = Math.min(100, Math.round((c / max) * 100));
  if (val) val.textContent = `${c}/${max}`;
  if (fill) fill.style.width = pct + '%';
  root.classList.add('is-active');
  root.classList.toggle('is-warn', c >= Math.ceil(max * 0.8) && !limitReached);
  root.classList.toggle('is-limit', !!limitReached);
  root.title = limitReached
    ? `Limite atingido (${c}/${max}). Use Continuar para seguir.`
    : `Turno ${c} de ${max} neste pedido`;
  if (btn) {
    btn.disabled = false;
    btn.classList.toggle('is-pulse', !!limitReached);
  }
}

