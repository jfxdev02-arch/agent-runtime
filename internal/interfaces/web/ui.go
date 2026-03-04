package web

func getIndexHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Agent Runtime</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
:root{--bg0:#060a10;--bg1:#0a0e17;--bg2:#111827;--bg3:#1a2332;--bg4:#1f2b3d;--bg-in:#0d1420;--brd:#1e2d3d;--brd-a:#3b82f6;--t1:#e2e8f0;--t2:#94a3b8;--t3:#64748b;--ac:#3b82f6;--ac2:#8b5cf6;--acg:rgba(59,130,246,.3);--ok:#10b981;--warn:#f59e0b;--err:#ef4444;--g1:linear-gradient(135deg,#3b82f6,#8b5cf6);--g2:linear-gradient(135deg,#06b6d4,#3b82f6);--sh:0 8px 32px rgba(0,0,0,.4);--r:12px;--tr:all .3s cubic-bezier(.4,0,.2,1)}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Inter',sans-serif;background:var(--bg1);color:var(--t1);height:100vh;overflow:hidden;display:flex}
::-webkit-scrollbar{width:6px}::-webkit-scrollbar-track{background:transparent}::-webkit-scrollbar-thumb{background:var(--brd);border-radius:3px}
.sidebar{width:72px;background:var(--bg2);border-right:1px solid var(--brd);display:flex;flex-direction:column;align-items:center;padding:16px 0;gap:8px;z-index:10}
.logo{width:44px;height:44px;background:var(--g1);border-radius:14px;display:flex;align-items:center;justify-content:center;font-weight:700;font-size:18px;margin-bottom:20px;box-shadow:0 4px 15px var(--acg)}
.nav-btn{width:48px;height:48px;border:none;background:0;color:var(--t3);border-radius:12px;cursor:pointer;display:flex;align-items:center;justify-content:center;transition:var(--tr);font-size:20px}
.nav-btn:hover{background:var(--bg3);color:var(--t1)}.nav-btn.active{background:var(--ac);color:#fff;box-shadow:0 4px 15px var(--acg)}
.main{flex:1;display:flex;flex-direction:column;overflow:hidden}
.page{display:none;flex:1;flex-direction:column;overflow:hidden}.page.active{display:flex}
.hdr{padding:20px 28px;border-bottom:1px solid var(--brd);display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}
.hdr h1{font-size:20px;font-weight:600;background:var(--g1);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
.badge{font-size:11px;padding:4px 10px;border-radius:20px;font-weight:500}
.badge-on{background:rgba(16,185,129,.15);color:var(--ok)}
.badge-active{background:rgba(59,130,246,.15);color:var(--ac)}
.badge-paused{background:rgba(245,158,11,.15);color:var(--warn)}
.badge-done{background:rgba(16,185,129,.15);color:var(--ok)}
.badge-archived{background:rgba(100,116,139,.15);color:var(--t3)}
.btn{border:none;border-radius:var(--r);font-weight:600;font-size:13px;cursor:pointer;transition:var(--tr);padding:8px 18px;display:inline-flex;align-items:center;gap:6px}
.btn-primary{background:var(--g1);color:#fff}.btn-primary:hover{transform:translateY(-1px);box-shadow:var(--sh)}
.btn-sm{padding:6px 12px;font-size:12px;border-radius:8px}
.btn-ghost{background:var(--bg3);color:var(--t2);border:1px solid var(--brd)}.btn-ghost:hover{border-color:var(--ac);color:var(--t1)}
.btn-danger{background:rgba(239,68,68,.15);color:var(--err);border:1px solid rgba(239,68,68,.2)}.btn-danger:hover{background:rgba(239,68,68,.25)}
.chat-c{flex:1;display:flex;flex-direction:column;overflow:hidden}
.msgs{flex:1;overflow-y:auto;padding:24px 28px;display:flex;flex-direction:column;gap:16px}
.msg{max-width:80%;padding:14px 18px;border-radius:16px;font-size:14px;line-height:1.6;animation:fadeIn .3s ease;word-wrap:break-word;white-space:pre-wrap}
@keyframes fadeIn{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:translateY(0)}}
.msg-user{align-self:flex-end;background:var(--ac);color:#fff;border-bottom-right-radius:4px}
.msg-assistant{align-self:flex-start;background:var(--bg3);border:1px solid var(--brd);border-bottom-left-radius:4px}
.msg-time{font-size:10px;color:var(--t3);margin-top:6px;opacity:.7}.msg-user .msg-time{color:rgba(255,255,255,.6)}
.chat-in{padding:16px 28px 24px;border-top:1px solid var(--brd);display:flex;gap:12px;align-items:flex-end}
.chat-in textarea{flex:1;background:var(--bg-in);border:1px solid var(--brd);color:var(--t1);border-radius:var(--r);padding:14px 18px;font-family:'Inter',sans-serif;font-size:14px;resize:none;height:52px;max-height:120px;transition:var(--tr);outline:0}
.chat-in textarea:focus{border-color:var(--ac);box-shadow:0 0 0 3px var(--acg)}
.send-btn{width:52px;height:52px;background:var(--g1);border:none;border-radius:var(--r);color:#fff;font-size:20px;cursor:pointer;transition:var(--tr);display:flex;align-items:center;justify-content:center;flex-shrink:0}
.send-btn:hover{transform:scale(1.05)}.send-btn:disabled{opacity:.5;cursor:not-allowed;transform:none}
.scroll{flex:1;overflow-y:auto;padding:24px 28px}
.sec{margin-bottom:28px}.sec h2{font-size:14px;font-weight:600;text-transform:uppercase;letter-spacing:1px;color:var(--t3);margin-bottom:16px}
.card{background:var(--bg3);border:1px solid var(--brd);border-radius:var(--r);padding:16px 20px;margin-bottom:12px;transition:var(--tr)}
.card:hover{border-color:var(--brd-a)}
.card label{display:block;font-size:13px;font-weight:500;color:var(--t2);margin-bottom:8px}
.card input,.card select,.card textarea{width:100%;background:var(--bg-in);border:1px solid var(--brd);color:var(--t1);border-radius:8px;padding:10px 14px;font-family:'JetBrains Mono',monospace;font-size:13px;outline:0;transition:var(--tr)}
.card input:focus,.card select:focus,.card textarea:focus{border-color:var(--ac);box-shadow:0 0 0 3px var(--acg)}
.card textarea{font-family:'Inter',sans-serif;resize:vertical;min-height:60px}
.log-e{background:var(--bg3);border:1px solid var(--brd);border-radius:var(--r);padding:14px 18px;margin-bottom:10px;cursor:pointer;transition:var(--tr)}
.log-e:hover{border-color:var(--brd-a)}.log-h{display:flex;align-items:center;gap:10px}
.log-tool{font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:500;color:var(--ac)}
.log-s{font-size:11px;padding:2px 8px;border-radius:10px;font-weight:600}
.log-ok{background:rgba(16,185,129,.15);color:var(--ok)}.log-err{background:rgba(239,68,68,.15);color:var(--err)}
.log-t{font-size:11px;color:var(--t3);margin-left:auto}
.log-d{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--t2);line-height:1.5;max-height:0;overflow:hidden;transition:max-height .3s;white-space:pre-wrap;word-break:break-all}
.log-e.exp .log-d{max-height:400px;overflow-y:auto;padding-top:10px;border-top:1px solid var(--brd);margin-top:8px}
.st-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:16px;margin-bottom:28px}
.st-c{background:var(--bg3);border:1px solid var(--brd);border-radius:var(--r);padding:20px;transition:var(--tr)}
.st-c:hover{border-color:var(--brd-a);transform:translateY(-2px);box-shadow:var(--sh)}
.st-l{font-size:12px;font-weight:500;text-transform:uppercase;letter-spacing:.5px;color:var(--t3);margin-bottom:8px}
.st-v{font-size:28px;font-weight:700;background:var(--g2);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
.st-s{font-size:12px;color:var(--t2);margin-top:4px}
.proj-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:16px;padding:24px 28px;overflow-y:auto}
.proj-card{background:var(--bg3);border:1px solid var(--brd);border-radius:var(--r);padding:20px;transition:var(--tr);cursor:pointer}
.proj-card:hover{border-color:var(--brd-a);transform:translateY(-2px);box-shadow:var(--sh)}
.proj-name{font-size:16px;font-weight:600;margin-bottom:6px}
.proj-path{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--t3);margin-bottom:12px;word-break:break-all}
.proj-meta{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:12px}
.proj-tech{font-size:11px;padding:3px 8px;border-radius:6px;background:rgba(139,92,246,.15);color:var(--ac2)}
.proj-branch{font-size:11px;padding:3px 8px;border-radius:6px;background:rgba(6,182,212,.15);color:#06b6d4}
.modal-bg{position:fixed;inset:0;background:rgba(0,0,0,.6);backdrop-filter:blur(4px);z-index:100;display:none;align-items:center;justify-content:center}
.modal-bg.show{display:flex}
.modal{background:var(--bg2);border:1px solid var(--brd);border-radius:16px;width:90%;max-width:700px;max-height:85vh;display:flex;flex-direction:column;box-shadow:0 20px 60px rgba(0,0,0,.5);animation:fadeIn .3s}
.modal-hdr{padding:20px 24px;border-bottom:1px solid var(--brd);display:flex;align-items:center;justify-content:space-between}
.modal-hdr h2{font-size:18px;font-weight:600}
.modal-close{background:0;border:0;color:var(--t3);font-size:24px;cursor:pointer}.modal-close:hover{color:var(--t1)}
.modal-body{flex:1;overflow-y:auto;padding:20px 24px}
.modal-tabs{display:flex;gap:4px;padding:0 24px;border-bottom:1px solid var(--brd)}
.modal-tab{padding:12px 16px;border:0;background:0;color:var(--t3);font-size:13px;font-weight:500;cursor:pointer;border-bottom:2px solid transparent;transition:var(--tr)}
.modal-tab:hover{color:var(--t1)}.modal-tab.active{color:var(--ac);border-bottom-color:var(--ac)}
.tab-panel{display:none}.tab-panel.active{display:block}
.git-pre{background:var(--bg0);border:1px solid var(--brd);border-radius:8px;padding:12px;font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--t2);white-space:pre-wrap;word-break:break-all;max-height:200px;overflow-y:auto;margin-bottom:12px}
.git-actions{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:16px}
.spinner{display:inline-block;width:18px;height:18px;border:2px solid rgba(255,255,255,.3);border-radius:50%;border-top-color:#fff;animation:spin .8s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.toast{position:fixed;bottom:24px;right:24px;padding:12px 24px;border-radius:var(--r);font-size:14px;font-weight:500;animation:slideUp .3s;z-index:1000;box-shadow:var(--sh)}
.toast-ok{background:var(--ok);color:#fff}.toast-err{background:var(--err);color:#fff}
@keyframes slideUp{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}
@media(max-width:768px){.sidebar{width:56px}.msgs{padding:16px}.msg{max-width:90%}.st-grid{grid-template-columns:1fr 1fr}.proj-grid{grid-template-columns:1fr}}
</style>
</head>
<body>
<aside class="sidebar">
  <div class="logo" id="logoLetter">A</div>
  <button class="nav-btn active" onclick="showPage('chat',this)" title="Chat">💬</button>
  <button class="nav-btn" onclick="showPage('projects',this)" title="Projects">📁</button>
  <button class="nav-btn" onclick="showPage('settings',this)" title="Settings">⚙️</button>
  <button class="nav-btn" onclick="showPage('logs',this)" title="Logs">📋</button>
  <button class="nav-btn" onclick="showPage('status',this)" title="Status">📊</button>
</aside>
<main class="main">
  <div id="page-chat" class="page active">
    <div class="hdr"><h1 id="chatTitle">Chat</h1><span class="badge badge-on">● Online</span></div>
    <div class="chat-c"><div class="msgs" id="messages"></div>
      <div class="chat-in"><textarea id="chatInput" placeholder="" rows="1" onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
        <button class="send-btn" id="sendBtn" onclick="sendMsg()">➤</button></div>
    </div>
  </div>
  <div id="page-projects" class="page">
    <div class="hdr"><h1 id="projTitle">Projects</h1>
      <div style="display:flex;gap:8px"><button class="btn btn-ghost" onclick="scanProjects()" id="btnScan">🔍 Scan</button><button class="btn btn-primary" onclick="showAddProject()" id="btnNewProj">+ New</button></div>
    </div>
    <div class="proj-grid" id="projGrid"></div>
  </div>
  <div id="page-settings" class="page">
    <div class="hdr"><h1 id="setTitle">Settings</h1><button class="btn btn-primary" onclick="saveSettings()" id="btnSave">Save</button></div>
    <div class="scroll">
      <div class="sec"><h2>LLM / API</h2>
        <div class="card"><label>Endpoint</label><input type="text" id="set-zai_endpoint"></div>
        <div class="card"><label>API Key</label><input type="password" id="set-zai_api_key"></div>
        <div class="card"><label>Model</label><input type="text" id="set-model"></div>
      </div>
      <div class="sec"><h2>Telegram</h2>
        <div class="card"><label>Bot Token</label><input type="password" id="set-telegram_token"></div>
        <div class="card"><label>Allow ID</label><input type="text" id="set-telegram_allow_id"></div>
      </div>
      <div class="sec"><h2>GitHub</h2>
        <div class="card"><label>Personal Access Token</label><input type="password" id="set-github_token"></div>
        <div class="card"><label>Username</label><input type="text" id="set-github_username"></div>
      </div>
      <div class="sec"><h2>Runtime</h2>
        <div class="card"><label>Agent Name</label><input type="text" id="set-agent_name"></div>
        <div class="card"><label id="lblLang">Language</label>
          <select id="set-language"><option value="en">English</option><option value="pt-BR">Português (Brasil)</option><option value="es">Español</option><option value="fr">Français</option><option value="de">Deutsch</option><option value="ja">日本語</option><option value="zh">中文</option></select>
        </div>
        <div class="card"><label>Workspace Root</label><input type="text" id="set-workspace_root"></div>
        <div class="card"><label>Max History</label><input type="number" id="set-max_history"></div>
        <div class="card"><label>Max Turns</label><input type="number" id="set-max_turns"></div>
      </div>
      <div class="sec"><h2 id="secUpdate">System Update</h2>
        <div class="card" style="position:relative">
          <label id="lblCurrentVer">Current Version</label>
          <div style="font-family:'JetBrains Mono',monospace;font-size:15px;font-weight:600;margin-bottom:12px" id="currentVerDisplay">—</div>
          <div style="display:flex;gap:8px;flex-wrap:wrap">
            <button class="btn btn-ghost" id="btnCheckUpdate" onclick="checkForUpdates()">🔍 <span id="btnCheckUpdateText">Check for Updates</span></button>
            <button class="btn btn-primary" id="btnApplyUpdate" onclick="applyUpdate()" style="display:none">⬇ <span id="btnApplyUpdateText">Update Now</span></button>
          </div>
          <div id="updateResult" style="margin-top:12px;display:none">
            <div id="updateInfo" class="git-pre" style="margin-bottom:8px"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div id="page-logs" class="page">
    <div class="hdr"><h1 id="logTitle">Tool Logs</h1><button class="btn btn-ghost" onclick="loadLogs()" id="btnRefreshLogs">Refresh</button></div>
    <div class="scroll" id="logsC"></div>
  </div>
  <div id="page-status" class="page">
    <div class="hdr"><h1 id="statTitle">System Status</h1><button class="btn btn-ghost" onclick="loadStatus()" id="btnRefreshStatus">Refresh</button></div>
    <div class="scroll" id="statusC"></div>
  </div>
</main>
<div class="modal-bg" id="projModal"><div class="modal">
  <div class="modal-hdr"><h2 id="modalTitle">Project</h2><button class="modal-close" onclick="closeModal()">&times;</button></div>
  <div class="modal-tabs">
    <button class="modal-tab active" onclick="showTab('overview',this)" id="tabOverview">Overview</button>
    <button class="modal-tab" onclick="showTab('git',this)">Git</button>
    <button class="modal-tab" onclick="showTab('notes',this)" id="tabNotes">Notes</button>
  </div>
  <div class="modal-body">
    <div class="tab-panel active" id="tab-overview">
      <div class="card"><label id="lblName">Name</label><input type="text" id="proj-name"></div>
      <div class="card"><label id="lblDesc">Description</label><input type="text" id="proj-desc"></div>
      <div class="card"><label>Status</label><select id="proj-status"><option value="active">Active</option><option value="paused">Paused</option><option value="done">Done</option><option value="archived">Archived</option></select></div>
      <div style="display:flex;gap:8px;margin-top:12px">
        <button class="btn btn-primary" onclick="saveProject()" id="btnSaveProj">Save</button>
        <button class="btn btn-danger" onclick="deleteProject()" id="btnDelProj">Delete</button>
      </div>
    </div>
    <div class="tab-panel" id="tab-git">
      <div class="git-actions">
        <button class="btn btn-ghost btn-sm" onclick="gitAction('pull')">⬇ Pull</button>
        <button class="btn btn-ghost btn-sm" onclick="gitAction('push')">⬆ Push</button>
        <button class="btn btn-ghost btn-sm" onclick="gitAction('init')">🆕 Git Init</button>
      </div>
      <div class="card"><label>Current Branch</label><div class="git-pre" id="git-branch">-</div></div>
      <div class="card"><label>Branches</label><div class="git-pre" id="git-branches">-</div></div>
      <div class="card" style="margin-bottom:8px"><label>Commit Message</label><input type="text" id="git-commit-msg" placeholder="Describe your changes"></div>
      <button class="btn btn-primary btn-sm" onclick="gitAction('commit')" style="margin-bottom:16px">✔ Commit All</button>
      <div class="card"><label>Branch Name</label><input type="text" id="git-branch-name" placeholder="feature/my-branch"></div>
      <div style="display:flex;gap:8px;margin-top:8px">
        <button class="btn btn-ghost btn-sm" onclick="gitAction('new_branch')">+ New Branch</button>
        <button class="btn btn-ghost btn-sm" onclick="gitAction('checkout')">↩ Checkout</button>
      </div>
      <div class="card" style="margin-top:16px"><label>Status</label><div class="git-pre" id="git-status">-</div></div>
      <div class="card"><label>Recent Commits</label><div class="git-pre" id="git-log">-</div></div>
      <div class="card"><label>Remote</label><div class="git-pre" id="git-remote">-</div></div>
    </div>
    <div class="tab-panel" id="tab-notes">
      <div class="card"><label>Project Notes (highlights, tasks, status)</label>
        <textarea id="proj-notes" rows="12" style="min-height:200px" placeholder="# Highlights&#10;- ...&#10;&#10;# TODO&#10;- ...&#10;&#10;# Status&#10;..."></textarea>
      </div>
      <button class="btn btn-primary" onclick="saveProject()">Save Notes</button>
    </div>
  </div>
</div></div>
<div class="modal-bg" id="addModal"><div class="modal" style="max-width:500px">
  <div class="modal-hdr"><h2 id="addTitle">Add Project</h2><button class="modal-close" onclick="closeAddModal()">&times;</button></div>
  <div class="modal-body">
    <div class="card"><label>Name</label><input type="text" id="add-name" placeholder="My Project"></div>
    <div class="card"><label>System Path</label><input type="text" id="add-path" placeholder="/home/user/project"></div>
    <div class="card"><label>Description</label><input type="text" id="add-desc" placeholder="Project description"></div>
    <div class="card"><label>Tech Stack</label><input type="text" id="add-tech" placeholder="Go, React, Python..."></div>
    <button class="btn btn-primary" onclick="addProject()" style="margin-top:12px" id="btnAddProj">Add</button>
  </div>
</div></div>
<script>
let currentProj=null, appCfg={agent_name:'Agent',language:'en'};
const esc=t=>(t||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/\n/g,'<br>');
const escPre=t=>(t||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');

// i18n
const i18n={
  en:{chat:'Chat',projects:'Projects',settings:'Settings',logs:'Tool Logs',status:'System Status',save:'Save',refresh:'Refresh',scan:'🔍 Scan Workspace',newProj:'+ New Project',addProj:'Add Project',add:'Add',overview:'Overview',notes:'Notes',name:'Name',desc:'Description',delete:'Delete',welcome:'Hello! I am your agentic assistant. How can I help?',placeholder:'Send a message...',noProj:'No projects yet. Use "Scan Workspace" or add manually.',noLogs:'No logs recorded yet.',saved:'Saved!',deleted:'Deleted!',added:'Project added!',scanned:' projects found!',confirmDel:'Delete project ',sysUpdate:'System Update',currentVer:'Current Version',checkUpdate:'Check for Updates',applyUpdate:'Update Now',upToDate:'✅ You are up to date!',updateAvail:'🆕 New version available: ',updating:'Updating... please wait...',updateOk:'✅ Update applied! Restarting...',updateFail:'❌ Update failed',noRelease:'No release notes',relNotes:'Release Notes'},
  'pt-BR':{chat:'Chat',projects:'Projetos',settings:'Configurações',logs:'Logs de Tools',status:'Status do Sistema',save:'Salvar',refresh:'Atualizar',scan:'🔍 Escanear Workspace',newProj:'+ Novo Projeto',addProj:'Adicionar Projeto',add:'Adicionar',overview:'Visão Geral',notes:'Notas',name:'Nome',desc:'Descrição',delete:'Excluir',welcome:'Olá! Sou seu assistente agêntico. Como posso ajudar?',placeholder:'Envie uma mensagem...',noProj:'Nenhum projeto. Use "Escanear Workspace" ou adicione manualmente.',noLogs:'Nenhum log registrado.',saved:'Salvo!',deleted:'Excluído!',added:'Projeto adicionado!',scanned:' projetos encontrados!',confirmDel:'Excluir projeto ',sysUpdate:'Atualização do Sistema',currentVer:'Versão Atual',checkUpdate:'Buscar Atualizações',applyUpdate:'Atualizar Agora',upToDate:'✅ Você está atualizado!',updateAvail:'🆕 Nova versão disponível: ',updating:'Atualizando... aguarde...',updateOk:'✅ Atualização aplicada! Reiniciando...',updateFail:'❌ Falha na atualização',noRelease:'Sem notas de versão',relNotes:'Notas da Versão'},
  es:{chat:'Chat',projects:'Proyectos',settings:'Configuración',logs:'Registros',status:'Estado del Sistema',save:'Guardar',refresh:'Actualizar',scan:'🔍 Escanear',newProj:'+ Nuevo',addProj:'Agregar Proyecto',add:'Agregar',overview:'General',notes:'Notas',name:'Nombre',desc:'Descripción',delete:'Eliminar',welcome:'¡Hola! Soy tu asistente agéntico. ¿Cómo puedo ayudar?',placeholder:'Envía un mensaje...',noProj:'Sin proyectos. Escanea o agrega manualmente.',noLogs:'Sin registros.',saved:'¡Guardado!',deleted:'¡Eliminado!',added:'¡Proyecto agregado!',scanned:' proyectos encontrados!',confirmDel:'Eliminar proyecto ',sysUpdate:'Actualización del Sistema',currentVer:'Versión Actual',checkUpdate:'Buscar Actualizaciones',applyUpdate:'Actualizar Ahora',upToDate:'✅ ¡Estás al día!',updateAvail:'🆕 Nueva versión disponible: ',updating:'Actualizando... espere...',updateOk:'✅ ¡Actualización aplicada! Reiniciando...',updateFail:'❌ Error en la actualización',noRelease:'Sin notas de versión',relNotes:'Notas de Versión'}
};

function t(key){const lang=i18n[appCfg.language]||i18n.en;return lang[key]||i18n.en[key]||key}

function applyI18n(){
  document.getElementById('chatTitle').textContent=appCfg.agent_name+' — '+t('chat');
  document.getElementById('projTitle').textContent=t('projects');
  document.getElementById('setTitle').textContent=t('settings');
  document.getElementById('logTitle').textContent=t('logs');
  document.getElementById('statTitle').textContent=t('status');
  document.getElementById('btnSave').textContent=t('save');
  document.getElementById('btnScan').textContent=t('scan');
  document.getElementById('btnNewProj').textContent=t('newProj');
  document.getElementById('btnRefreshLogs').textContent=t('refresh');
  document.getElementById('btnRefreshStatus').textContent=t('refresh');
  document.getElementById('tabOverview').textContent=t('overview');
  document.getElementById('tabNotes').textContent=t('notes');
  document.getElementById('btnSaveProj').textContent=t('save');
  document.getElementById('btnDelProj').textContent=t('delete');
  document.getElementById('addTitle').textContent=t('addProj');
  document.getElementById('btnAddProj').textContent=t('add');
  document.getElementById('chatInput').placeholder=t('placeholder');
  document.getElementById('logoLetter').textContent=appCfg.agent_name.charAt(0).toUpperCase();
  document.title=appCfg.agent_name+' — Agent Runtime';
  document.getElementById('secUpdate').textContent=t('sysUpdate');
  document.getElementById('lblCurrentVer').textContent=t('currentVer');
  document.getElementById('btnCheckUpdateText').textContent=t('checkUpdate');
  document.getElementById('btnApplyUpdateText').textContent=t('applyUpdate');
  document.getElementById('currentVerDisplay').textContent=appCfg.version||'—';
}

async function loadAppConfig(){
  try{const r=await fetch('/api/app-config');appCfg=await r.json();applyI18n()}catch(e){}
}

function showPage(n,btn){document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));document.querySelectorAll('.nav-btn').forEach(b=>b.classList.remove('active'));document.getElementById('page-'+n).classList.add('active');btn.classList.add('active');if(n==='logs')loadLogs();if(n==='status')loadStatus();if(n==='settings')loadSettings();if(n==='projects')loadProjects()}
function toast(m,type='ok'){const d=document.createElement('div');d.className='toast toast-'+type;d.textContent=m;document.body.appendChild(d);setTimeout(()=>d.remove(),3000)}

async function sendMsg(){const i=document.getElementById('chatInput'),txt=i.value.trim();if(!txt)return;appendMsg('user',txt);i.value='';i.style.height='52px';const b=document.getElementById('sendBtn');b.disabled=true;b.innerHTML='<span class="spinner"></span>';try{const r=await fetch('/api/chat',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({session_id:'web-default',message:txt})});const d=await r.json();appendMsg('assistant',d.reply)}catch(e){appendMsg('assistant','Error: '+e.message)}b.disabled=false;b.innerHTML='➤'}
function appendMsg(r,txt){const c=document.getElementById('messages'),d=document.createElement('div');d.className='msg msg-'+r;const tm=new Date().toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});d.innerHTML=esc(txt)+'<div class="msg-time">'+tm+'</div>';c.appendChild(d);c.scrollTop=c.scrollHeight}

async function loadSettings(){try{const r=await fetch('/api/settings'),d=await r.json();['zai_endpoint','zai_api_key','model','telegram_token','telegram_allow_id','workspace_root','max_history','max_turns','github_token','github_username','agent_name','language'].forEach(f=>{const e=document.getElementById('set-'+f);if(e&&d[f]){if(e.tagName==='SELECT')e.value=d[f];else e.value=d[f]}})}catch(e){}}
async function saveSettings(){const s={};['zai_endpoint','zai_api_key','model','telegram_token','telegram_allow_id','workspace_root','max_history','max_turns','github_token','github_username','agent_name','language'].forEach(f=>{const e=document.getElementById('set-'+f);if(e&&e.value)s[f]=e.value});try{await fetch('/api/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(s)});if(s.agent_name)appCfg.agent_name=s.agent_name;if(s.language)appCfg.language=s.language;applyI18n();toast(t('saved'))}catch(e){toast('Error: '+e.message,'err')}}

async function loadLogs(){try{const r=await fetch('/api/logs'),l=await r.json(),c=document.getElementById('logsC');if(!l||!l.length){c.innerHTML='<p style="color:var(--t3);text-align:center;padding:40px">'+t('noLogs')+'</p>';return}c.innerHTML=l.map(x=>'<div class="log-e" onclick="this.classList.toggle(\'exp\')"><div class="log-h"><span class="log-tool">'+escPre(x.tool_name)+'</span><span class="log-s log-'+(x.status==='OK'?'ok':'err')+'">'+x.status+'</span><span class="log-t">'+x.created_at+'</span></div><div class="log-d">INPUT:\n'+escPre((x.input||'').substring(0,300))+'\n\nOUTPUT:\n'+escPre((x.output||'').substring(0,500))+'</div></div>').join('')}catch(e){}}

async function loadStatus(){try{const r=await fetch('/api/status'),s=await r.json(),db=s.db_stats||{},u=Math.floor(s.uptime_seconds/3600)+'h '+Math.floor(s.uptime_seconds%3600/60)+'m';document.getElementById('statusC').innerHTML='<div class="st-grid">'+[['Hostname',s.hostname,s.os_arch],['Uptime',u,'Go '+s.go_version],['Memory',(s.mem_alloc_mb||0).toFixed(1)+' MB',(s.mem_sys_mb||0).toFixed(1)+' MB sys'],['Goroutines',s.goroutines,''],['Messages',db.total_messages||0,(db.total_sessions||0)+' sessions'],['Executions',db.total_tool_executions||0,(db.successful_executions||0)+' OK'],['Projects',db.total_projects||0,'']].map(x=>'<div class="st-c"><div class="st-l">'+x[0]+'</div><div class="st-v">'+x[1]+'</div><div class="st-s">'+x[2]+'</div></div>').join('')+'</div>'}catch(e){}}

async function loadProjects(){try{const r=await fetch('/api/projects'),p=await r.json(),c=document.getElementById('projGrid');if(!p||!p.length){c.innerHTML='<p style="color:var(--t3);text-align:center;padding:40px;grid-column:1/-1">'+t('noProj')+'</p>';return}c.innerHTML=p.map(x=>{const st=x.status||'active';const techs=(x.tech_stack||'').split(', ').filter(Boolean).map(t=>'<span class="proj-tech">'+t+'</span>').join('');const br=x.git_remote?'<span class="proj-branch">'+escPre(x.git_remote)+'</span>':'';return'<div class="proj-card" onclick="openProject('+x.id+')"><div class="proj-name">'+esc(x.name)+'</div><div class="proj-path">'+escPre(x.path)+'</div><div class="proj-meta">'+techs+br+'<span class="badge badge-'+st+'">'+st+'</span></div>'+(x.description?'<div style="font-size:13px;color:var(--t2)">'+esc(x.description)+'</div>':'')+'</div>'}).join('')}catch(e){}}

async function scanProjects(){toast('Scanning...');try{const r=await fetch('/api/projects/scan'),d=await r.json();toast(d.scanned+t('scanned'));loadProjects()}catch(e){toast('Error','err')}}
function showAddProject(){document.getElementById('addModal').classList.add('show')}
function closeAddModal(){document.getElementById('addModal').classList.remove('show')}
async function addProject(){const n=document.getElementById('add-name').value,p=document.getElementById('add-path').value;if(!n||!p){toast('Name and path required','err');return}try{await fetch('/api/projects',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:n,path:p,description:document.getElementById('add-desc').value,tech_stack:document.getElementById('add-tech').value})});closeAddModal();loadProjects();toast(t('added'))}catch(e){toast('Error','err')}}

async function openProject(id){try{const r=await fetch('/api/projects'),ps=await r.json(),p=ps.find(x=>x.id===id);if(!p)return;currentProj=p;document.getElementById('modalTitle').textContent=p.name;document.getElementById('proj-name').value=p.name;document.getElementById('proj-desc').value=p.description||'';document.getElementById('proj-status').value=p.status||'active';document.getElementById('proj-notes').value=p.notes||'';document.getElementById('projModal').classList.add('show');showTab('overview',document.querySelector('.modal-tab'));loadGitInfo(id)}catch(e){}}
function closeModal(){document.getElementById('projModal').classList.remove('show');currentProj=null}
function showTab(name,btn){document.querySelectorAll('.tab-panel').forEach(p=>p.classList.remove('active'));document.querySelectorAll('.modal-tab').forEach(b=>b.classList.remove('active'));document.getElementById('tab-'+name).classList.add('active');btn.classList.add('active')}

async function saveProject(){if(!currentProj)return;try{await fetch('/api/projects',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:currentProj.id,name:document.getElementById('proj-name').value,description:document.getElementById('proj-desc').value,status:document.getElementById('proj-status').value,notes:document.getElementById('proj-notes').value})});toast(t('saved'));loadProjects()}catch(e){toast('Error','err')}}
async function deleteProject(){if(!currentProj||!confirm(t('confirmDel')+currentProj.name+'?'))return;try{await fetch('/api/projects',{method:'DELETE',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:currentProj.id})});closeModal();loadProjects();toast(t('deleted'))}catch(e){toast('Error','err')}}

async function loadGitInfo(id){try{const r=await fetch('/api/projects/git?id='+id),d=await r.json();document.getElementById('git-branch').textContent=d.branch||'(no git)';document.getElementById('git-branches').textContent=d.branches||'-';document.getElementById('git-status').textContent=d.status||'Clean';document.getElementById('git-log').textContent=d.log||'-';document.getElementById('git-remote').textContent=d.remote||'(no remote)'}catch(e){}}
async function gitAction(action){if(!currentProj)return;const body={id:currentProj.id,action};if(action==='commit')body.message=document.getElementById('git-commit-msg').value;if(action==='new_branch'||action==='checkout')body.branch=document.getElementById('git-branch-name').value;if((action==='new_branch'||action==='checkout')&&!body.branch){toast('Branch name required','err');return}try{const r=await fetch('/api/projects/git/action',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const d=await r.json();toast(action+': OK');loadGitInfo(currentProj.id);if(d.output)document.getElementById('git-status').textContent=d.output}catch(e){toast('Git error: '+e.message,'err')}}

// --- Update System ---
let pendingUpdateInfo=null;
async function checkForUpdates(){
  const btn=document.getElementById('btnCheckUpdate');
  const origHTML=btn.innerHTML;
  btn.disabled=true;btn.innerHTML='<span class="spinner"></span> '+t('checkUpdate');
  const res=document.getElementById('updateResult');
  const info=document.getElementById('updateInfo');
  try{
    const r=await fetch('/api/update/check');
    const d=await r.json();
    if(d.error){res.style.display='block';info.textContent='Error: '+d.error;info.style.borderColor='var(--err)';toast(d.error,'err');return}
    res.style.display='block';
    document.getElementById('currentVerDisplay').textContent=d.current_version;
    if(d.update_available){
      pendingUpdateInfo=d;
      info.innerHTML=t('updateAvail')+'<strong style="color:var(--ok)">'+escPre(d.latest_version)+'</strong>';
      if(d.release_notes){info.innerHTML+='\n\n<strong>'+t('relNotes')+':</strong>\n'+escPre(d.release_notes)}
      if(d.published_at){info.innerHTML+='\n\n📅 '+d.published_at.substring(0,10)}
      info.style.borderColor='var(--ok)';
      document.getElementById('btnApplyUpdate').style.display='inline-flex';
      toast(t('updateAvail')+d.latest_version)
    }else{
      pendingUpdateInfo=null;
      info.textContent=t('upToDate')+'\n'+t('currentVer')+': '+d.current_version;
      info.style.borderColor='var(--ok)';
      document.getElementById('btnApplyUpdate').style.display='none';
      toast(t('upToDate'))
    }
  }catch(e){res.style.display='block';info.textContent='Error: '+e.message;info.style.borderColor='var(--err)';toast('Error: '+e.message,'err')}
  finally{btn.disabled=false;btn.innerHTML=origHTML}
}
async function applyUpdate(){
  if(!confirm(t('applyUpdate')+'?'))return;
  const btn=document.getElementById('btnApplyUpdate');
  const info=document.getElementById('updateInfo');
  btn.disabled=true;btn.innerHTML='<span class="spinner"></span> '+t('updating');
  info.textContent=t('updating');
  info.style.borderColor='var(--warn)';
  try{
    const r=await fetch('/api/update/apply',{method:'POST'});
    const d=await r.json();
    if(d.success){
      info.textContent=t('updateOk')+'\n\n'+d.message+'\n\n'+d.output;
      info.style.borderColor='var(--ok)';
      toast(t('updateOk'));
      // Auto-reload after service restarts
      setTimeout(()=>{location.reload()},8000)
    }else{
      info.textContent=t('updateFail')+'\n\n'+d.message+'\n\n'+(d.output||'');
      info.style.borderColor='var(--err)';
      toast(t('updateFail'),'err');
      btn.disabled=false;btn.innerHTML='⬇ '+t('applyUpdate')
    }
  }catch(e){info.textContent='Connection lost — service may be restarting...';info.style.borderColor='var(--warn)';setTimeout(()=>{location.reload()},8000)}
}

document.getElementById('chatInput').addEventListener('input',function(){this.style.height='52px';this.style.height=Math.min(this.scrollHeight,120)+'px'});

// Init
loadAppConfig().then(()=>{appendMsg('assistant',t('welcome'))});
</script>
</body></html>`
}
