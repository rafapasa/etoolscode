import { pushActivity, setChatTurnCounter } from './activitylog.js';
import { saveCurrentChat, state } from './state.js';

function escapeHtml(s){ return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
function formatContent(t){
  if(!t) return '<span class="muted">...</span>';
  return escapeHtml(t)
    .replace(/```([\s\S]*?)```/g,(m,c)=>`<div style="background:#0F172A;color:#E2E8F0;padding:10px;border-radius:8px;margin:6px 0;font-family:monospace;font-size:12px;overflow:auto;border-left:3px solid #16A34A;white-space:pre-wrap">${escapeHtml(c.trim())}</div>`)
    .replace(/\n/g,'<br>');
}

function chatBox(){ return document.getElementById('chatMessages'); }
function nearBottom(el, px=80){
  if(!el) return true;
  return (el.scrollHeight - el.scrollTop - el.clientHeight) <= px;
}
function stickScroll(el, pinned){
  if(pinned && el) el.scrollTop = el.scrollHeight;
}

function removeChatActions(){
  document.querySelectorAll('.chat-limit-actions').forEach(n => n.remove());
}

function showLimitActions(assistantDiv){
  removeChatActions();
  const wrap=document.createElement('div');
  wrap.className='chat-limit-actions message assistant';
  wrap.innerHTML=`
    <div class="chat-limit-banner">⚠ Limite de rodadas deste pedido. A operação pode estar incompleta.</div>
    <div class="chat-limit-btns">
      <button type="button" class="btn primary" data-act="continue">▶ Continuar</button>
      <button type="button" class="btn-small" data-act="resumo">Resumo</button>
    </div>`;
  const box=chatBox();
  if(box) box.appendChild(wrap);
  else if(assistantDiv && assistantDiv.parentElement) assistantDiv.parentElement.appendChild(wrap);
  stickScroll(box, true);
  wrap.querySelector('[data-act="continue"]')?.addEventListener('click', () => {
    removeChatActions();
    continueChatWork();
  });
  wrap.querySelector('[data-act="resumo"]')?.addEventListener('click', () => {
    removeChatActions();
    resumoChatWork();
  });
  // destaca botão do rodapé
  const fb=document.getElementById('btnChatContinueFooter');
  if(fb){ fb.disabled=false; fb.classList.add('is-pulse'); }
}

export function continueChatWork(){
  return sendMessageWithPreset('Continue de onde parou. Não refaça o que já está concluído; complete apenas o que falta.');
}

export function resumoChatWork(){
  return sendMessageWithPreset('Faça um resumo objetivo do que já foi feito neste chat, o que ficou pendente e o próximo passo recomendado. Não use tools.');
}

async function sendMessageWithPreset(text){
  const input=document.getElementById('chatInput');
  if(input){
    input.value=text;
    if(typeof resizeChatInput==='function') resizeChatInput();
  }
  await sendMessage();
}

// rodapé / atalhos
if(typeof window!=='undefined'){
  window.__etoolsChatContinue = () => continueChatWork();
}


export function addMessage(role, content, returnEl=false, forceScroll=false){
  const c=chatBox();
  if(!c) return null;
  const pinned = forceScroll || role==='user' || nearBottom(c);
  const e=c.querySelector('.empty-state'); if(e) e.remove();
  const div=document.createElement('div');
  div.className='message '+role;
  div.innerHTML=role==='user'?escapeHtml(content):formatContent(content);
  c.appendChild(div);
  stickScroll(c, pinned);
  if(returnEl) return div;
  return null;
}

function toolKind(name){
  if(name==='list_files' || name==='list_dir') return 'list';
  if(name==='write_file' || name==='search_replace' || name==='apply_patch') return 'write';
  if(name==='run_command') return 'command';
  if(name && name.startsWith('git_')) return 'git';
  return 'read';
}

/** Tools vão para o painel de log (não poluem o chat). */
function addToolLine(data){
  const kind = data.kind || toolKind(data.name || '');
  const ok = data.ok !== false;
  let path = (data.path || '').trim();
  let detail = (data.detail || data.error || '').trim();
  if (!path && detail) path = detail.split('\n')[0];
  if (!path) path = data.name || '(tool)';
  pushActivity({
    kind,
    ok,
    name: data.name || '',
    path,
    detail: detail && detail !== path ? detail : (data.error || ''),
    error: data.error || '',
    source: 'chat',
  });
}


function formatWhen(iso){
  if(!iso) return '';
  const d=new Date(iso);
  if(Number.isNaN(d.getTime())) return '';
  return d.toLocaleString('pt-BR', {day:'2-digit', month:'2-digit', hour:'2-digit', minute:'2-digit'});
}

export async function loadChatList(){
  const listEl=document.getElementById('chatList');
  if(!listEl) return;
  try{
    const res=await fetch('/api/chat');
    if(!res.ok) throw new Error(await res.text());
    const data=await res.json();
    let chats=data.chats || [];
    if(state.currentProject){
      chats=chats.filter(c => c.project === state.currentProject);
    }
    if(!chats.length){
      listEl.innerHTML='<div class="empty-state" style="margin:16px 8px;color:#94A3B8">Nenhum chat. Use + novo.</div>';
      return;
    }
    listEl.innerHTML=chats.map(c => {
      const raw=(c.title || 'Novo chat').replace(/\s+/g,' ').trim() || 'Novo chat';
      const title=escapeHtml(raw);
      const active=c.id===state.currentChatId ? ' selected' : '';
      const when=escapeHtml(formatWhen(c.updated_at));
      const payload=encodeURIComponent(JSON.stringify({kind:'chat', id:c.id, project:c.project||''}));
      return `<div class="file-item${active}" data-chat-id="${c.id}" title="${when}">
        <span class="file-label">
          <span class="file-icon json-icon" aria-hidden="true">{}</span>
          <span class="file-name">${title}</span>
        </span>
        <button type="button" class="kebab-btn" data-menu="${payload}" title="Ações" aria-label="Ações">
          <span></span><span></span><span></span>
        </button>
      </div>`;
    }).join('');
  }catch(e){
    listEl.innerHTML=`<div class="empty-state" style="margin:16px 8px;color:#fca5a5">${escapeHtml(e.message)}</div>`;
  }
}

export async function openChat(id){
  if(!id) return;
  const chatMessages=document.getElementById('chatMessages');
  try{
    const res=await fetch('/api/chat/'+encodeURIComponent(id));
    if(!res.ok) throw new Error(await res.text());
    const session=await res.json();
    state.currentChatId=session.id;
    if(session.project) state.currentProject=session.project;
    saveCurrentChat();
    renderSession(session);
    await loadChatList();
  }catch(e){
    state.currentChatId=null;
    saveCurrentChat();
    if(chatMessages) chatMessages.innerHTML=`<div class="empty-state">Não foi possível abrir o chat: ${escapeHtml(e.message)}</div>`;
    await loadChatList();
  }
}

function renderSession(session){
  const chatMessages=document.getElementById('chatMessages');
  if(!chatMessages) return;
  chatMessages.innerHTML='';
  const msgs=session.messages || [];
  if(!msgs.some(m => m.role==='user' || (m.role==='assistant' && m.content))){
    chatMessages.innerHTML=`<div class="message system">Chat ${(session.id||'').slice(0,8)} - ${escapeHtml(session.project||'')} ✅ Pronto</div>`;
    return;
  }
  for(const m of msgs){
    if(m.role==='system') continue;
    if(m.role==='tool'){
      const content = m.content || '';
      const ok = !content.startsWith('ERRO:') && !content.startsWith('erro');
      // tenta extrair path do conteúdo tipico: "Arquivo escrito: path" / "arquivo escrito: path"
      let path = '';
      const mPath = content.match(/(?:arquivo|Arquivo|search_replace ok:|lido|path)[:\s]+([^\s(]+)/i);
      if (mPath) path = mPath[1];
      addToolLine({name:m.name||'read_file', path: path || (ok ? '(histórico)' : content.slice(0,120)), detail: content.slice(0,200), ok});
      continue;
    }
    if(m.role==='assistant' && !m.content && m.tool_calls) continue;
    addMessage(m.role, m.content||'', false, true);
  }
  const box=chatBox();
  if(box) box.scrollTop=box.scrollHeight;
}

export async function createNewChat(){
  const chatMessages=document.getElementById('chatMessages');
  const sel=document.getElementById('projectSelector');
  if(sel && sel.value && !state.currentProject) state.currentProject=sel.value;
  if(!state.currentProject){
    if(chatMessages) chatMessages.innerHTML='<div class="empty-state">Selecione projeto no combo superior</div>';
    return null;
  }
  try{
    const res=await fetch('/api/chat/new',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({project:state.currentProject})});
    if(!res.ok){
      const err=await res.text();
      if(chatMessages) chatMessages.innerHTML=`<div class="message system" style="background:#3f1d1d;color:#fca5a5;border:1px solid #7f1d1d">Erro criar chat: ${escapeHtml(err)}</div>`;
      return null;
    }
    const data=await res.json();
    if(!data.id) return null;
    state.currentChatId=data.id;
    saveCurrentChat();
    if(chatMessages) chatMessages.innerHTML=`<div class="message system">Chat ${data.id.slice(0,8)} - ${state.currentProject} ✅ Pronto</div>`;
    await loadChatList();
    return data.id;
  }catch(e){
    if(chatMessages) chatMessages.innerHTML=`<div class="message system" style="background:#3f1d1d;color:#fca5a5;border:1px solid #7f1d1d">Erro: ${escapeHtml(e.message)}</div>`;
    return null;
  }
}

export async function sendMessage(){
  const input=document.getElementById('chatInput');
  const sel=document.getElementById('projectSelector');
  const text=input?input.value.trim():'';
  if(sel && sel.value && state.currentProject!==sel.value){
    state.currentProject=sel.value;
    localStorage.setItem('etools_current_project', state.currentProject);
    state.currentChatId=null;
  }
  if(!text) return;
  if(!state.currentProject){ alert('Selecione projeto no combo superior'); return; }
  if(!state.currentChatId){
    const newId=await createNewChat();
    if(!newId){ alert('Falha ao criar chat'); return; }
  }
  if(input){
    input.value='';
    resizeChatInput();
  }
  addMessage('user', text, false, true);
  const assistantDiv=addMessage('assistant','',true, true);
  let full='';
  try{
    const res=await fetch(`/api/chat/${state.currentChatId}/message`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:text})});
    if(!res.ok){
      const errText=await res.text();
      if(res.status===404){
        state.currentChatId=null;
        const nid=await createNewChat();
        if(nid) return sendMessageWithText(text, assistantDiv);
      }
      if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ Erro ${res.status}: ${escapeHtml(errText)}</span>`;
      return;
    }
    const reader=res.body.getReader(); const decoder=new TextDecoder(); let buffer='';
    while(true){
      const {done,value}=await reader.read(); if(done) break;
      buffer+=decoder.decode(value,{stream:true});
      const lines=buffer.split('\n\n'); buffer=lines.pop()||'';
      for(const line of lines){
        if(!line.startsWith('data: ')) continue;
        try{
          const data=JSON.parse(line.slice(6));
          if(data.type==='turn'){
            setChatTurnCounter(data.current, data.max, false);
          }
          else if(data.type==='delta'){
            const box=chatBox();
            const pinned=nearBottom(box);
            full+=data.content||'';
            if(assistantDiv) assistantDiv.innerHTML=formatContent(full);
            stickScroll(box, pinned);
          }
          else if(data.type==='tool'){ addToolLine(data); }
          else if(data.type==='error'){ if(assistantDiv) assistantDiv.innerHTML+=`<br><span style="color:#b91c1c">❌ ${escapeHtml(data.error)}</span>`; }
          else if(data.type==='done'){
            setChatTurnCounter(data.current||0, data.max||0, !!data.limit_reached);
            if(data.limit_reached){
              showLimitActions(assistantDiv);
            }
          }
        }catch{}
      }
    }
    await loadChatList();
  }catch(e){ if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ ${e.message}</span>`; }
}

async function sendMessageWithText(text, assistantDiv){
  if(!state.currentChatId) return;
  let full='';
  try{
    const res=await fetch(`/api/chat/${state.currentChatId}/message`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:text})});
    if(!res.ok){ if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ ${await res.text()}</span>`; return; }
    const reader=res.body.getReader(); const decoder=new TextDecoder(); let buffer='';
    while(true){
      const {done,value}=await reader.read(); if(done) break;
      buffer+=decoder.decode(value,{stream:true});
      const lines=buffer.split('\n\n'); buffer=lines.pop()||'';
      for(const line of lines){
        if(!line.startsWith('data: ')) continue;
        try{
          const data=JSON.parse(line.slice(6));
          if(data.type==='turn'){
            setChatTurnCounter(data.current, data.max, false);
          }
          else if(data.type==='delta'){
            const box=chatBox();
            const pinned=nearBottom(box);
            full+=data.content||'';
            if(assistantDiv) assistantDiv.innerHTML=formatContent(full);
            stickScroll(box, pinned);
          }
          else if(data.type==='tool'){ addToolLine(data); }
          else if(data.type==='error'){ if(assistantDiv) assistantDiv.innerHTML+=`<br><span style="color:#b91c1c">❌ ${escapeHtml(data.error)}</span>`; }
          else if(data.type==='done'){
            setChatTurnCounter(data.current||0, data.max||0, !!data.limit_reached);
            if(data.limit_reached){
              showLimitActions(assistantDiv);
            }
          }
        }catch{}
      }
    }
  }catch(e){ if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ ${e.message}</span>`; }
}

export function resizeChatInput(){
  const el=document.getElementById('chatInput');
  if(!el) return;
  if(!el.dataset.baseH){
    el.style.height='auto';
    el.dataset.baseH=String(Math.max(el.scrollHeight, 40));
  }
  const base=Number(el.dataset.baseH)||40;
  const max=base*4;
  el.style.height='auto';
  const next=Math.min(Math.max(el.scrollHeight, base), max);
  el.style.height=next+'px';
  el.style.overflowY=el.scrollHeight>max+1 ? 'auto' : 'hidden';
}

export async function renameChat(id){
  if(!id) return;
  const atual=document.querySelector(`[data-chat-id="${CSS.escape(id)}"] .file-name`);
  const nome=prompt('Novo nome do chat', atual?atual.textContent:'');
  if(nome===null) return;
  const title=nome.trim();
  if(!title) return;
  const res=await fetch('/api/chat/'+encodeURIComponent(id),{
    method:'PATCH',
    headers:{'Content-Type':'application/json'},
    body:JSON.stringify({title})
  });
  if(!res.ok){ alert(await res.text()); return; }
  await loadChatList();
}

export async function exportChatResumo(id){
  if(!id) return;
  const res=await fetch('/api/chat/'+encodeURIComponent(id)+'/resumo');
  if(!res.ok){ alert(await res.text()); return; }
  const blob=await res.blob();
  const url=URL.createObjectURL(blob);
  const a=document.createElement('a');
  a.href=url;
  a.download='resumo.md';
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export async function deleteChat(id){
  if(!id) return;
  if(!confirm('Excluir este chat?')) return;
  const res=await fetch('/api/chat/'+encodeURIComponent(id),{method:'DELETE'});
  if(!res.ok){ alert(await res.text()); return; }
  if(state.currentChatId===id){
    state.currentChatId=null;
    saveCurrentChat();
    const msgs=document.getElementById('chatMessages');
    if(msgs) msgs.innerHTML='<div class="empty-state">Chat excluído. Selecione outro ou clique em + novo.</div>';
  }
  await loadChatList();
}

export { escapeHtml, formatContent };
