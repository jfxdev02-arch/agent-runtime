package web

func getIndexHTML() string {
	return `<!DOCTYPE html>
<html lang="en" data-theme="nova-dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Agent Runtime</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
:root{--r:12px;--tr:all .3s cubic-bezier(.4,0,.2,1)}
[data-theme="nova-dark"]{--bg0:#060a10;--bg1:#0e1621;--bg2:#17212b;--bg3:#1e2c3a;--bg4:#242f3d;--bg-in:#0e1621;--brd:#2b3945;--brd-a:#2aabee;--t1:#e2e8f0;--t2:#8b9bab;--t3:#6c7c8c;--ac:#2aabee;--ac2:#6ab2f2;--acg:rgba(42,171,238,.3);--ok:#4fae4e;--warn:#e5a64e;--err:#e05555;--g1:linear-gradient(135deg,#2aabee,#6ab2f2);--g2:linear-gradient(135deg,#229ed9,#2aabee);--sh:0 8px 32px rgba(0,0,0,.4)}
[data-theme="nova-light"]{--bg0:#dae3ed;--bg1:#f0f2f5;--bg2:#ffffff;--bg3:#e8edf2;--bg4:#dce3ea;--bg-in:#ffffff;--brd:#c8d0d8;--brd-a:#2aabee;--t1:#1a2233;--t2:#5a6a7a;--t3:#7a8a9a;--ac:#2aabee;--ac2:#168acd;--acg:rgba(42,171,238,.18);--ok:#4fae4e;--warn:#d4941e;--err:#e05555;--g1:linear-gradient(135deg,#2aabee,#6ab2f2);--g2:linear-gradient(135deg,#229ed9,#2aabee);--sh:0 4px 20px rgba(0,0,0,.08)}
[data-theme="pulse-dark"]{--bg0:#0b141a;--bg1:#111b21;--bg2:#1a2730;--bg3:#233138;--bg4:#2a3942;--bg-in:#111b21;--brd:#2a3942;--brd-a:#00a884;--t1:#e9edef;--t2:#8696a0;--t3:#667781;--ac:#00a884;--ac2:#53bdeb;--acg:rgba(0,168,132,.3);--ok:#00a884;--warn:#f7c948;--err:#ea4335;--g1:linear-gradient(135deg,#00a884,#25d366);--g2:linear-gradient(135deg,#00a884,#53bdeb);--sh:0 8px 32px rgba(0,0,0,.4)}
[data-theme="pulse-light"]{--bg0:#d8ded3;--bg1:#f0f2f5;--bg2:#ffffff;--bg3:#e7ebe4;--bg4:#dce0d7;--bg-in:#ffffff;--brd:#c8cec2;--brd-a:#00a884;--t1:#111b21;--t2:#54656f;--t3:#667781;--ac:#00a884;--ac2:#0088cc;--acg:rgba(0,168,132,.18);--ok:#00a884;--warn:#e6a817;--err:#ea4335;--g1:linear-gradient(135deg,#00a884,#25d366);--g2:linear-gradient(135deg,#00a884,#53bdeb);--sh:0 4px 20px rgba(0,0,0,.08)}
[data-theme="eclipse-dark"]{--bg0:#1e1f22;--bg1:#2b2d31;--bg2:#313338;--bg3:#383a40;--bg4:#404249;--bg-in:#1e1f22;--brd:#3f4147;--brd-a:#5865f2;--t1:#f2f3f5;--t2:#b5bac1;--t3:#949ba4;--ac:#5865f2;--ac2:#eb459e;--acg:rgba(88,101,242,.3);--ok:#57f287;--warn:#fee75c;--err:#ed4245;--g1:linear-gradient(135deg,#5865f2,#eb459e);--g2:linear-gradient(135deg,#5865f2,#57f287);--sh:0 8px 32px rgba(0,0,0,.4)}
[data-theme="eclipse-light"]{--bg0:#dddee1;--bg1:#f2f3f5;--bg2:#ffffff;--bg3:#ebedef;--bg4:#e0e1e5;--bg-in:#ffffff;--brd:#d4d6da;--brd-a:#5865f2;--t1:#060607;--t2:#4e5058;--t3:#80848e;--ac:#5865f2;--ac2:#da3775;--acg:rgba(88,101,242,.18);--ok:#248046;--warn:#d4941e;--err:#da373c;--g1:linear-gradient(135deg,#5865f2,#eb459e);--g2:linear-gradient(135deg,#5865f2,#57f287);--sh:0 4px 20px rgba(0,0,0,.08)}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Inter',sans-serif;background:var(--bg1);color:var(--t1);height:100vh;overflow:hidden;display:flex}
::-webkit-scrollbar{width:6px}::-webkit-scrollbar-track{background:transparent}::-webkit-scrollbar-thumb{background:var(--brd);border-radius:3px}
.sidebar{width:72px;background:var(--bg2);border-right:1px solid var(--brd);display:flex;flex-direction:column;align-items:center;padding:16px 0;gap:8px;z-index:10}
.logo{width:44px;height:44px;background:var(--g1);border-radius:14px;display:flex;align-items:center;justify-content:center;font-weight:700;font-size:18px;margin-bottom:20px;box-shadow:0 4px 15px var(--acg);color:#fff}
.nav-btn{width:48px;height:48px;border:none;background:0;color:var(--t3);border-radius:12px;cursor:pointer;display:flex;align-items:center;justify-content:center;transition:var(--tr);font-size:20px}
.nav-btn:hover{background:var(--bg3);color:var(--t1)}.nav-btn.active{background:var(--ac);color:#fff;box-shadow:0 4px 15px var(--acg)}
.nav-btn svg{pointer-events:none}
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
.btn svg{pointer-events:none;flex-shrink:0}
.btn-primary{background:var(--g1);color:#fff}.btn-primary:hover{transform:translateY(-1px);box-shadow:var(--sh)}
.btn-sm{padding:6px 12px;font-size:12px;border-radius:8px}
.btn-ghost{background:var(--bg3);color:var(--t2);border:1px solid var(--brd)}.btn-ghost:hover{border-color:var(--ac);color:var(--t1)}
.btn-danger{background:rgba(239,68,68,.15);color:var(--err);border:1px solid rgba(239,68,68,.2)}.btn-danger:hover{background:rgba(239,68,68,.25)}
.chat-c{flex:1;display:flex;flex-direction:column;overflow:hidden}
.chat-layout{flex:1;display:flex;min-height:0}
.chat-sidebar{width:280px;min-width:240px;background:var(--bg2);border-right:1px solid var(--brd);display:flex;flex-direction:column}
.chat-sidebar-h{padding:14px;border-bottom:1px solid var(--brd)}
.chat-sessions{flex:1;overflow-y:auto;padding:10px}
.chat-session{width:100%;text-align:left;background:var(--bg3);border:1px solid var(--brd);color:var(--t1);border-radius:10px;padding:10px 12px;margin-bottom:8px;cursor:pointer;transition:var(--tr)}
.chat-session:hover{border-color:var(--brd-a)}
.chat-session.active{border-color:var(--ac);box-shadow:0 0 0 2px var(--acg)}
.chat-session-id{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--t3);margin-bottom:4px}
.chat-session-msg{font-size:12px;color:var(--t2);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.chat-session-empty{font-size:12px;color:var(--t3);padding:12px;text-align:center}
.chat-session{position:relative}
.chat-del{position:absolute;top:8px;right:8px;width:22px;height:22px;border:none;background:transparent;color:var(--t3);border-radius:6px;cursor:pointer;font-size:13px;display:flex;align-items:center;justify-content:center;opacity:0;transition:var(--tr)}
.chat-session:hover .chat-del{opacity:1}
.chat-del:hover{background:rgba(239,68,68,.2);color:var(--err)}
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
.theme-picker{position:fixed;left:80px;bottom:16px;background:var(--bg2);border:1px solid var(--brd);border-radius:var(--r);padding:16px;width:220px;box-shadow:var(--sh);z-index:50;display:none;animation:fadeIn .2s}
.theme-picker.show{display:block}
.theme-picker-hdr{display:flex;align-items:center;justify-content:space-between;margin-bottom:14px;font-size:13px;font-weight:600;color:var(--t1)}
.theme-mode-tgl{background:var(--bg3);border:1px solid var(--brd);border-radius:8px;width:34px;height:34px;display:flex;align-items:center;justify-content:center;cursor:pointer;color:var(--t2);transition:var(--tr)}.theme-mode-tgl:hover{border-color:var(--ac);color:var(--ac)}
.theme-opt{width:100%;display:flex;align-items:center;gap:10px;padding:10px 12px;border:1px solid var(--brd);background:var(--bg3);border-radius:10px;cursor:pointer;transition:var(--tr);margin-bottom:8px;color:var(--t1);font-size:13px;font-weight:500}
.theme-opt:last-child{margin-bottom:0}
.theme-opt:hover{border-color:var(--brd-a)}.theme-opt.active{border-color:var(--ac);box-shadow:0 0 0 2px var(--acg)}
.theme-prev{width:28px;height:28px;border-radius:8px;display:flex;overflow:hidden;flex-shrink:0}.theme-prev span{flex:1}
.pulse-prev span:nth-child(1){background:#00a884}.pulse-prev span:nth-child(2){background:#25d366}
.nova-prev span:nth-child(1){background:#2aabee}.nova-prev span:nth-child(2){background:#6ab2f2}
.eclipse-prev span:nth-child(1){background:#5865f2}.eclipse-prev span:nth-child(2){background:#eb459e}
@media(max-width:768px){.sidebar{width:56px}.msgs{padding:16px}.msg{max-width:90%}.st-grid{grid-template-columns:1fr 1fr}.proj-grid{grid-template-columns:1fr}.chat-sidebar{display:none}.theme-picker{left:64px}}
</style>
</head>
<body>
<aside class="sidebar">
  <div class="logo" id="logoLetter">A</div>
  <button class="nav-btn active" onclick="showPage('chat',this)" title="Chat"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg></button>
  <button class="nav-btn" onclick="showPage('projects',this)" title="Projects"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg></button>
  <button class="nav-btn" onclick="showPage('providers',this)" title="Providers"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg></button>
  <button class="nav-btn" onclick="showPage('settings',this)" title="Settings"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="21" x2="4" y2="14"/><line x1="4" y1="10" x2="4" y2="3"/><line x1="12" y1="21" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="3"/><line x1="20" y1="21" x2="20" y2="16"/><line x1="20" y1="12" x2="20" y2="3"/><line x1="1" y1="14" x2="7" y2="14"/><line x1="9" y1="8" x2="15" y2="8"/><line x1="17" y1="16" x2="23" y2="16"/></svg></button>
  <button class="nav-btn" onclick="showPage('logs',this)" title="Logs"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><line x1="10" y1="9" x2="8" y2="9"/></svg></button>
  <button class="nav-btn" onclick="showPage('status',this)" title="Status"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg></button>
  <div style="flex:1"></div>
  <button class="nav-btn" onclick="toggleThemePicker(event)" title="Theme" id="themeTglBtn"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.93 0 1.5-.75 1.5-1.5 0-.4-.17-.78-.4-1.08s-.4-.68-.4-1.12c0-.92.75-1.68 1.68-1.68H16a6 6 0 0 0 6-6c0-5.52-4.48-10-10-10z"/><circle cx="8" cy="12" r="1.5" fill="currentColor" stroke="none"/><circle cx="12" cy="7.5" r="1.5" fill="currentColor" stroke="none"/><circle cx="16" cy="12" r="1.5" fill="currentColor" stroke="none"/></svg></button>
</aside>
<main class="main">
  <div id="page-chat" class="page active">
    <div class="hdr"><h1 id="chatTitle">Chat</h1><div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap"><span class="badge badge-on" id="chatSessionBadge">Online</span>
      <select id="chatThinkLevel" onchange="updateSessionSetting()" style="background:var(--bg3);color:var(--t2);border:1px solid var(--brd);border-radius:8px;padding:4px 8px;font-size:11px;cursor:pointer" title="Think Level"><option value="off">Off</option><option value="low">Low</option><option value="medium" selected>Medium</option><option value="high">High</option></select>
      <select id="chatModelSelect" onchange="updateSessionSetting()" style="background:var(--bg3);color:var(--t2);border:1px solid var(--brd);border-radius:8px;padding:4px 8px;font-size:11px;cursor:pointer" title="Model"><option value="">Default</option></select>
      <label style="display:flex;align-items:center;gap:4px;font-size:11px;color:var(--t3);cursor:pointer" title="Verbose Mode"><input type="checkbox" id="chatVerbose" onchange="updateSessionSetting()"> Verbose</label>
      <button class="btn btn-ghost btn-sm" onclick="compactSession()" title="Compact session (summarize history)"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 14 10 14 10 20"/><polyline points="20 10 14 10 14 4"/><line x1="14" y1="10" x2="21" y2="3"/><line x1="3" y1="21" x2="10" y2="14"/></svg> <span id="btnCompactText">Compact</span></button>
    </div></div>
    <div class="chat-layout">
      <aside class="chat-sidebar">
        <div class="chat-sidebar-h"><button class="btn btn-primary" style="width:100%" id="btnNewChat" onclick="newChat()"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg> <span id="btnNewChatText">New Chat</span></button></div>
        <div class="chat-sessions" id="chatSessions"></div>
      </aside>
      <div class="chat-c"><div class="msgs" id="messages"></div>
        <div class="chat-in"><textarea id="chatInput" placeholder="" rows="1" onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
          <button class="send-btn" id="sendBtn" onclick="sendMsg()"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg></button></div>
      </div>
    </div>
  </div>
  <div id="page-projects" class="page">
    <div class="hdr"><h1 id="projTitle">Projects</h1>
      <div style="display:flex;gap:8px"><button class="btn btn-ghost" onclick="scanProjects()" id="btnScan"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg> <span id="btnScanText">Scan</span></button><button class="btn btn-primary" onclick="showAddProject()" id="btnNewProj"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg> <span id="btnNewProjText">New</span></button></div>
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
          <select id="set-language"><option value="en">English</option><option value="pt-BR">Portugues (Brasil)</option><option value="es">Espanol</option><option value="fr">Francais</option><option value="de">Deutsch</option><option value="ja">Japanese</option><option value="zh">Chinese</option></select>
        </div>
        <div class="card"><label>Workspace Root</label><input type="text" id="set-workspace_root"></div>
        <div class="card"><label>Max History</label><input type="number" id="set-max_history"></div>
        <div class="card"><label>Max Turns</label><input type="number" id="set-max_turns"></div>
      </div>
      <div class="sec"><h2 id="secUpdate">System Update</h2>
        <div class="card" style="position:relative">
          <label id="lblCurrentVer">Current Version</label>
          <div style="font-family:'JetBrains Mono',monospace;font-size:15px;font-weight:600;margin-bottom:12px" id="currentVerDisplay">--</div>
          <div style="display:flex;gap:8px;flex-wrap:wrap">
            <button class="btn btn-ghost" id="btnCheckUpdate" onclick="checkForUpdates()"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg> <span id="btnCheckUpdateText">Check for Updates</span></button>
            <button class="btn btn-primary" id="btnApplyUpdate" onclick="applyUpdate()" style="display:none"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg> <span id="btnApplyUpdateText">Update Now</span></button>
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
  <div id="page-providers" class="page">
    <div class="hdr"><h1 id="provTitle">Model Providers</h1><button class="btn btn-ghost" onclick="loadProviderStatus()" id="btnRefreshProv">Refresh</button></div>
    <div class="scroll" id="provC">
      <div class="sec"><h2>Configured Providers</h2>
        <p style="color:var(--t3);font-size:13px;margin-bottom:16px">Set <code style="background:var(--bg3);padding:2px 6px;border-radius:4px;font-family:'JetBrains Mono',monospace;font-size:12px">MODELS</code> env var to configure multiple providers with automatic failover.<br>
        Format: <code style="background:var(--bg3);padding:2px 6px;border-radius:4px;font-family:'JetBrains Mono',monospace;font-size:12px">id:name:endpoint:key:model:priority||...</code></p>
        <div id="providerList"></div>
      </div>
    </div>
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
        <button class="btn btn-ghost btn-sm" onclick="gitAction('pull')"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg> Pull</button>
        <button class="btn btn-ghost btn-sm" onclick="gitAction('push')"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg> Push</button>
        <button class="btn btn-ghost btn-sm" onclick="gitAction('init')"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg> Git Init</button>
      </div>
      <div class="card"><label>Current Branch</label><div class="git-pre" id="git-branch">-</div></div>
      <div class="card"><label>Branches</label><div class="git-pre" id="git-branches">-</div></div>
      <div class="card" style="margin-bottom:8px"><label>Commit Message</label><input type="text" id="git-commit-msg" placeholder="Describe your changes"></div>
      <button class="btn btn-primary btn-sm" onclick="gitAction('commit')" style="margin-bottom:16px"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg> Commit All</button>
      <div class="card"><label>Branch Name</label><input type="text" id="git-branch-name" placeholder="feature/my-branch"></div>
      <div style="display:flex;gap:8px;margin-top:8px">
        <button class="btn btn-ghost btn-sm" onclick="gitAction('new_branch')"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg> New Branch</button>
        <button class="btn btn-ghost btn-sm" onclick="gitAction('checkout')"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 14 4 9 9 4"/><path d="M20 20v-7a4 4 0 0 0-4-4H4"/></svg> Checkout</button>
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
<div class="modal-bg" id="onboardModal"><div class="modal" style="max-width:600px">
  <div class="modal-hdr"><h2><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/><path d="M12 15l-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/><path d="M9 12H4s.55-3.03 2-4c1.62-1.08 3 0 3 0"/><path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-3 0-3"/></svg> Welcome -- Setup Wizard</h2><button class="modal-close" onclick="closeOnboarding()">&times;</button></div>
  <div class="modal-body">
    <div id="onb-step-1" class="onb-step">
      <h3 style="margin-bottom:12px;color:var(--ac)">Step 1: Agent Identity</h3>
      <p style="color:var(--t2);margin-bottom:16px;font-size:13px">Give your agent a name and choose the interface language.</p>
      <div class="card"><label>Agent Name</label><input type="text" id="onb-name" placeholder="Cronos" value="Cronos"></div>
      <div class="card"><label>Language</label><select id="onb-lang"><option value="en">English</option><option value="pt-BR">Portugues (Brasil)</option><option value="es">Espanol</option><option value="fr">Francais</option><option value="de">Deutsch</option><option value="ja">Japanese</option><option value="zh">Chinese</option></select></div>
      <button class="btn btn-primary" onclick="onbNext(2)" style="margin-top:12px">Next &rarr;</button>
    </div>
    <div id="onb-step-2" class="onb-step" style="display:none">
      <h3 style="margin-bottom:12px;color:var(--ac)">Step 2: LLM Provider</h3>
      <p style="color:var(--t2);margin-bottom:16px;font-size:13px">Configure your primary LLM API. Any OpenAI-compatible endpoint works (OpenAI, Anthropic, Together, ZhipuAI, Ollama, etc.).</p>
      <div class="card"><label>API Endpoint</label><input type="text" id="onb-endpoint" placeholder="https://api.openai.com/v1/chat/completions"></div>
      <div class="card"><label>API Key</label><input type="password" id="onb-apikey" placeholder="sk-..."></div>
      <div class="card"><label>Model Name</label><input type="text" id="onb-model" placeholder="gpt-4o" value="gpt-4o"></div>
      <div id="onb-validate-result" style="display:none;margin-bottom:12px;padding:12px;border-radius:8px;font-size:13px"></div>
      <div style="display:flex;gap:8px;margin-top:12px">
        <button class="btn btn-ghost" onclick="onbNext(1)">&larr; Back</button>
        <button class="btn btn-ghost" onclick="onbValidate()" id="onb-validate-btn"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg> Test Connection</button>
        <button class="btn btn-primary" onclick="onbNext(3)">Next &rarr;</button>
      </div>
    </div>
    <div id="onb-step-3" class="onb-step" style="display:none">
      <h3 style="margin-bottom:12px;color:var(--ac)">Step 3: Workspace</h3>
      <p style="color:var(--t2);margin-bottom:16px;font-size:13px">Where should the agent work? This is the root directory for file operations and project scanning.</p>
      <div class="card"><label>Workspace Root</label><input type="text" id="onb-workspace" placeholder="/home/user/projects" value="."></div>
      <div style="display:flex;gap:8px;margin-top:12px">
        <button class="btn btn-ghost" onclick="onbNext(2)">&larr; Back</button>
        <button class="btn btn-primary" onclick="onbNext(4)">Next &rarr;</button>
      </div>
    </div>
    <div id="onb-step-4" class="onb-step" style="display:none">
      <h3 style="margin-bottom:12px;color:var(--ac)">Step 4: Telegram (Optional)</h3>
      <p style="color:var(--t2);margin-bottom:16px;font-size:13px">Connect a Telegram bot to chat with your agent from anywhere. Skip if you only need the web interface.</p>
      <div class="card"><label>Bot Token (from @BotFather)</label><input type="password" id="onb-tg-token" placeholder="123456:ABC-DEF..."></div>
      <div class="card"><label>Your Telegram User ID</label><input type="text" id="onb-tg-id" placeholder="12345678"></div>
      <div style="display:flex;gap:8px;margin-top:12px">
        <button class="btn btn-ghost" onclick="onbNext(3)">&larr; Back</button>
        <button class="btn btn-primary" onclick="onbFinish()"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg> Finish Setup</button>
      </div>
    </div>
  </div>
</div></div>
<div class="theme-picker" id="themePicker">
  <div class="theme-picker-hdr">
    <span id="themePickerTitle">Theme</span>
    <button class="theme-mode-tgl" onclick="toggleThemeMode()" id="themeModeTgl" title="Toggle light/dark"></button>
  </div>
  <button class="theme-opt" data-theme="pulse" onclick="setThemeBase('pulse')">
    <span class="theme-prev pulse-prev"><span></span><span></span></span>
    Pulse
  </button>
  <button class="theme-opt" data-theme="nova" onclick="setThemeBase('nova')">
    <span class="theme-prev nova-prev"><span></span><span></span></span>
    Nova
  </button>
  <button class="theme-opt" data-theme="eclipse" onclick="setThemeBase('eclipse')">
    <span class="theme-prev eclipse-prev"><span></span><span></span></span>
    Eclipse
  </button>
</div>
<script>
// --- SVG Icon constants for dynamic JS usage ---
var ICN={
  send:'<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>',
  search:'<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>',
  trash:'<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>',
  download:'<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
  sun:'<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>',
  moon:'<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>'
};

// --- Theme System ---
var themeBase=localStorage.getItem('themeBase')||'nova';
var themeMode=localStorage.getItem('themeMode')||'dark';
function applyTheme(){
  var full=themeBase+'-'+themeMode;
  document.documentElement.setAttribute('data-theme',full);
  localStorage.setItem('themeBase',themeBase);
  localStorage.setItem('themeMode',themeMode);
  var tgl=document.getElementById('themeModeTgl');
  if(tgl) tgl.innerHTML=themeMode==='dark'?ICN.sun:ICN.moon;
  document.querySelectorAll('.theme-opt').forEach(function(o){o.classList.toggle('active',o.getAttribute('data-theme')===themeBase)});
}
function setThemeBase(b){themeBase=b;applyTheme()}
function toggleThemeMode(){themeMode=themeMode==='dark'?'light':'dark';applyTheme()}
function toggleThemePicker(e){e&&e.stopPropagation();document.getElementById('themePicker').classList.toggle('show')}
document.addEventListener('click',function(e){var p=document.getElementById('themePicker');var b=document.getElementById('themeTglBtn');if(p&&p.classList.contains('show')&&!p.contains(e.target)&&e.target!==b&&!b.contains(e.target)){p.classList.remove('show')}});
applyTheme();

// --- App globals ---
var currentProj=null, appCfg={agent_name:'Agent',language:'en'};
var currentSessionId='web-default';
var esc=function(t){return(t||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/\n/g,'<br>')};
var escPre=function(t){return(t||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')};

// i18n
var i18n={
  en:{chat:'Chat',projects:'Projects',settings:'Settings',logs:'Tool Logs',status:'System Status',save:'Save',refresh:'Refresh',scan:'Scan Workspace',newProj:'New Project',addProj:'Add Project',add:'Add',overview:'Overview',notes:'Notes',name:'Name',desc:'Description',delete:'Delete',welcome:'Hello! I am your agentic assistant. How can I help?',placeholder:'Send a message...',noProj:'No projects yet. Use "Scan Workspace" or add manually.',noLogs:'No logs recorded yet.',saved:'Saved!',deleted:'Deleted!',added:'Project added!',scanned:' projects found!',confirmDel:'Delete project ',sysUpdate:'System Update',currentVer:'Current Version',checkUpdate:'Check for Updates',applyUpdate:'Update Now',upToDate:'Up to date!',updateAvail:'New version available: ',updating:'Updating... please wait...',updateOk:'Update applied! Restarting...',updateFail:'Update failed',noRelease:'No release notes',relNotes:'Release Notes',newChat:'New Chat',chatHistoryEmpty:'No previous chats',session:'Session',compact:'Compact',theme:'Theme'},
  'pt-BR':{chat:'Chat',projects:'Projetos',settings:'Configuracoes',logs:'Logs de Tools',status:'Status do Sistema',save:'Salvar',refresh:'Atualizar',scan:'Escanear Workspace',newProj:'Novo Projeto',addProj:'Adicionar Projeto',add:'Adicionar',overview:'Visao Geral',notes:'Notas',name:'Nome',desc:'Descricao',delete:'Excluir',welcome:'Ola! Sou seu assistente agentico. Como posso ajudar?',placeholder:'Envie uma mensagem...',noProj:'Nenhum projeto. Use "Escanear Workspace" ou adicione manualmente.',noLogs:'Nenhum log registrado.',saved:'Salvo!',deleted:'Excluido!',added:'Projeto adicionado!',scanned:' projetos encontrados!',confirmDel:'Excluir projeto ',sysUpdate:'Atualizacao do Sistema',currentVer:'Versao Atual',checkUpdate:'Buscar Atualizacoes',applyUpdate:'Atualizar Agora',upToDate:'Voce esta atualizado!',updateAvail:'Nova versao disponivel: ',updating:'Atualizando... aguarde...',updateOk:'Atualizacao aplicada! Reiniciando...',updateFail:'Falha na atualizacao',noRelease:'Sem notas de versao',relNotes:'Notas da Versao',newChat:'Novo Chat',chatHistoryEmpty:'Sem chats anteriores',session:'Sessao',compact:'Compactar',theme:'Tema'},
  es:{chat:'Chat',projects:'Proyectos',settings:'Configuracion',logs:'Registros',status:'Estado del Sistema',save:'Guardar',refresh:'Actualizar',scan:'Escanear',newProj:'Nuevo',addProj:'Agregar Proyecto',add:'Agregar',overview:'General',notes:'Notas',name:'Nombre',desc:'Descripcion',delete:'Eliminar',welcome:'Hola! Soy tu asistente agentico. Como puedo ayudar?',placeholder:'Envia un mensaje...',noProj:'Sin proyectos. Escanea o agrega manualmente.',noLogs:'Sin registros.',saved:'Guardado!',deleted:'Eliminado!',added:'Proyecto agregado!',scanned:' proyectos encontrados!',confirmDel:'Eliminar proyecto ',sysUpdate:'Actualizacion del Sistema',currentVer:'Version Actual',checkUpdate:'Buscar Actualizaciones',applyUpdate:'Actualizar Ahora',upToDate:'Estas al dia!',updateAvail:'Nueva version disponible: ',updating:'Actualizando... espere...',updateOk:'Actualizacion aplicada! Reiniciando...',updateFail:'Error en la actualizacion',noRelease:'Sin notas de version',relNotes:'Notas de Version',newChat:'Nuevo Chat',chatHistoryEmpty:'Sin chats anteriores',session:'Sesion',compact:'Compactar',theme:'Tema'}
};

function t(key){var lang=i18n[appCfg.language]||i18n.en;return lang[key]||i18n.en[key]||key}

function applyI18n(){
  document.getElementById('chatTitle').textContent=appCfg.agent_name+' -- '+t('chat');
  document.getElementById('projTitle').textContent=t('projects');
  document.getElementById('setTitle').textContent=t('settings');
  document.getElementById('logTitle').textContent=t('logs');
  document.getElementById('statTitle').textContent=t('status');
  document.getElementById('btnSave').textContent=t('save');
  document.getElementById('btnScanText').textContent=t('scan');
  document.getElementById('btnNewProjText').textContent=t('newProj');
  document.getElementById('btnRefreshLogs').textContent=t('refresh');
  document.getElementById('btnRefreshStatus').textContent=t('refresh');
  document.getElementById('tabOverview').textContent=t('overview');
  document.getElementById('tabNotes').textContent=t('notes');
  document.getElementById('btnSaveProj').textContent=t('save');
  document.getElementById('btnDelProj').textContent=t('delete');
  document.getElementById('addTitle').textContent=t('addProj');
  document.getElementById('btnAddProj').textContent=t('add');
  document.getElementById('chatInput').placeholder=t('placeholder');
  document.getElementById('btnNewChatText').textContent=t('newChat');
  document.getElementById('btnCompactText').textContent=t('compact');
  document.getElementById('logoLetter').textContent=appCfg.agent_name.charAt(0).toUpperCase();
  document.title=appCfg.agent_name+' -- Agent Runtime';
  document.getElementById('secUpdate').textContent=t('sysUpdate');
  document.getElementById('lblCurrentVer').textContent=t('currentVer');
  document.getElementById('btnCheckUpdateText').textContent=t('checkUpdate');
  document.getElementById('btnApplyUpdateText').textContent=t('applyUpdate');
  document.getElementById('currentVerDisplay').textContent=appCfg.version||'--';
  document.getElementById('themePickerTitle').textContent=t('theme');
}

async function loadAppConfig(){
  try{var r=await fetch('/api/app-config');appCfg=await r.json();applyI18n()}catch(e){}
}

function showPage(n,btn){document.querySelectorAll('.page').forEach(function(p){p.classList.remove('active')});document.querySelectorAll('.nav-btn').forEach(function(b){b.classList.remove('active')});document.getElementById('page-'+n).classList.add('active');btn.classList.add('active');if(n==='logs')loadLogs();if(n==='status')loadStatus();if(n==='settings')loadSettings();if(n==='projects')loadProjects();if(n==='providers')loadProviderStatus()}
function toast(m,type){var d=document.createElement('div');d.className='toast toast-'+(type||'ok');d.textContent=m;document.body.appendChild(d);setTimeout(function(){d.remove()},3000)}

async function sendMsg(){var i=document.getElementById('chatInput'),txt=i.value.trim();if(!txt)return;appendMsg('user',txt);i.value='';i.style.height='52px';var b=document.getElementById('sendBtn');b.disabled=true;b.innerHTML='<span class="spinner"></span>';try{var r=await fetch('/api/chat',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({session_id:currentSessionId,message:txt})});var d=await r.json();currentSessionId=d.session_id||currentSessionId;updateSessionBadge();appendMsg('assistant',d.reply);loadChatSessions()}catch(e){appendMsg('assistant','Error: '+e.message)}b.disabled=false;b.innerHTML=ICN.send}

function updateSessionBadge(){document.getElementById('chatSessionBadge').textContent=t('session')+': '+currentSessionId}

function renderChatSessions(list){var c=document.getElementById('chatSessions');if(!list||!list.length){c.innerHTML='<div class="chat-session-empty">'+t('chatHistoryEmpty')+'</div>';return}c.innerHTML=list.map(function(s){var sid=(s.session_id||'').replace(/'/g,'');var msg=escPre((s.last_message||'').slice(0,80));var active=sid===currentSessionId?' active':'';return '<div class="chat-session'+active+'" onclick="openSession(\''+sid+'\')">'+'<button class="chat-del" onclick="event.stopPropagation();deleteChat(\''+sid+'\')" title="Delete chat">'+ICN.trash+'</button>'+'<div class="chat-session-id">'+escPre(sid)+'</div><div class="chat-session-msg">'+(msg||'-')+'</div></div>'}).join('')}

async function deleteChat(sessionID){
  if(!confirm('Delete this chat? This cannot be undone.'))return;
  try{
    var r=await fetch('/api/chat/delete',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({session_id:sessionID})});
    var d=await r.json();
    if(d.error){toast('Error: '+d.error,'err');return}
    toast('Chat deleted');
    if(sessionID===currentSessionId){await newChat()}else{await loadChatSessions()}
  }catch(e){toast('Error: '+e.message,'err')}
}

async function loadChatSessions(){try{var r=await fetch('/api/chats?prefix=web-&limit=40');var sessions=await r.json();renderChatSessions(sessions)}catch(e){}}

async function loadChatHistory(sessionID){try{var r=await fetch('/api/chat/history?session_id='+encodeURIComponent(sessionID));var msgs=await r.json();var c=document.getElementById('messages');c.innerHTML='';if(!msgs||!msgs.length){appendMsg('assistant',t('welcome'));return}msgs.forEach(function(m){if(m.role==='user'||m.role==='assistant'){appendMsg(m.role,m.content)}})}catch(e){appendMsg('assistant','Error: '+e.message)}}

async function openSession(sessionID){currentSessionId=sessionID;updateSessionBadge();await loadChatHistory(sessionID);await loadChatSessions();await loadSessionSettings()}

async function newChat(){try{var r=await fetch('/api/chat/new',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({prefix:'web'})});var d=await r.json();currentSessionId=d.session_id;updateSessionBadge();document.getElementById('messages').innerHTML='';appendMsg('assistant',t('welcome'));loadChatSessions()}catch(e){toast('Error: '+e.message,'err')}}
function appendMsg(r,txt){var c=document.getElementById('messages'),d=document.createElement('div');d.className='msg msg-'+r;var tm=new Date().toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});d.innerHTML=esc(txt)+'<div class="msg-time">'+tm+'</div>';c.appendChild(d);c.scrollTop=c.scrollHeight}

async function loadSettings(){try{var r=await fetch('/api/settings'),d=await r.json();['zai_endpoint','zai_api_key','model','telegram_token','telegram_allow_id','workspace_root','max_history','max_turns','github_token','github_username','agent_name','language'].forEach(function(f){var e=document.getElementById('set-'+f);if(e&&d[f]){if(e.tagName==='SELECT')e.value=d[f];else e.value=d[f]}})}catch(e){}}
async function saveSettings(){var s={};['zai_endpoint','zai_api_key','model','telegram_token','telegram_allow_id','workspace_root','max_history','max_turns','github_token','github_username','agent_name','language'].forEach(function(f){var e=document.getElementById('set-'+f);if(e&&e.value)s[f]=e.value});try{await fetch('/api/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(s)});if(s.agent_name)appCfg.agent_name=s.agent_name;if(s.language)appCfg.language=s.language;applyI18n();toast(t('saved'))}catch(e){toast('Error: '+e.message,'err')}}

async function loadLogs(){try{var r=await fetch('/api/logs'),l=await r.json(),c=document.getElementById('logsC');if(!l||!l.length){c.innerHTML='<p style="color:var(--t3);text-align:center;padding:40px">'+t('noLogs')+'</p>';return}c.innerHTML=l.map(function(x){return '<div class="log-e" onclick="this.classList.toggle(\'exp\')"><div class="log-h"><span class="log-tool">'+escPre(x.tool_name)+'</span><span class="log-s log-'+(x.status==='OK'?'ok':'err')+'">'+x.status+'</span><span class="log-t">'+x.created_at+'</span></div><div class="log-d">INPUT:\n'+escPre((x.input||'').substring(0,300))+'\n\nOUTPUT:\n'+escPre((x.output||'').substring(0,500))+'</div></div>'}).join('')}catch(e){}}

async function loadStatus(){try{var r=await fetch('/api/status'),s=await r.json(),db=s.db_stats||{},u=Math.floor(s.uptime_seconds/3600)+'h '+Math.floor(s.uptime_seconds%3600/60)+'m';document.getElementById('statusC').innerHTML='<div class="st-grid">'+[['Hostname',s.hostname,s.os_arch],['Uptime',u,'Go '+s.go_version],['Memory',(s.mem_alloc_mb||0).toFixed(1)+' MB',(s.mem_sys_mb||0).toFixed(1)+' MB sys'],['Goroutines',s.goroutines,''],['Messages',db.total_messages||0,(db.total_sessions||0)+' sessions'],['Executions',db.total_tool_executions||0,(db.successful_executions||0)+' OK'],['Projects',db.total_projects||0,'']].map(function(x){return '<div class="st-c"><div class="st-l">'+x[0]+'</div><div class="st-v">'+x[1]+'</div><div class="st-s">'+x[2]+'</div></div>'}).join('')+'</div>'}catch(e){}}

async function loadProjects(){try{var r=await fetch('/api/projects'),p=await r.json(),c=document.getElementById('projGrid');if(!p||!p.length){c.innerHTML='<p style="color:var(--t3);text-align:center;padding:40px;grid-column:1/-1">'+t('noProj')+'</p>';return}c.innerHTML=p.map(function(x){var st=x.status||'active';var techs=(x.tech_stack||'').split(', ').filter(Boolean).map(function(t){return '<span class="proj-tech">'+t+'</span>'}).join('');var br=x.git_remote?'<span class="proj-branch">'+escPre(x.git_remote)+'</span>':'';return'<div class="proj-card" onclick="openProject('+x.id+')"><div class="proj-name">'+esc(x.name)+'</div><div class="proj-path">'+escPre(x.path)+'</div><div class="proj-meta">'+techs+br+'<span class="badge badge-'+st+'">'+st+'</span></div>'+(x.description?'<div style="font-size:13px;color:var(--t2)">'+esc(x.description)+'</div>':'')+'</div>'}).join('')}catch(e){}}

async function scanProjects(){toast('Scanning...');try{var r=await fetch('/api/projects/scan'),d=await r.json();toast(d.scanned+t('scanned'));loadProjects()}catch(e){toast('Error','err')}}
function showAddProject(){document.getElementById('addModal').classList.add('show')}
function closeAddModal(){document.getElementById('addModal').classList.remove('show')}
async function addProject(){var n=document.getElementById('add-name').value,p=document.getElementById('add-path').value;if(!n||!p){toast('Name and path required','err');return}try{await fetch('/api/projects',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:n,path:p,description:document.getElementById('add-desc').value,tech_stack:document.getElementById('add-tech').value})});closeAddModal();loadProjects();toast(t('added'))}catch(e){toast('Error','err')}}

async function openProject(id){try{var r=await fetch('/api/projects'),ps=await r.json(),p=ps.find(function(x){return x.id===id});if(!p)return;currentProj=p;document.getElementById('modalTitle').textContent=p.name;document.getElementById('proj-name').value=p.name;document.getElementById('proj-desc').value=p.description||'';document.getElementById('proj-status').value=p.status||'active';document.getElementById('proj-notes').value=p.notes||'';document.getElementById('projModal').classList.add('show');showTab('overview',document.querySelector('.modal-tab'));loadGitInfo(id)}catch(e){}}
function closeModal(){document.getElementById('projModal').classList.remove('show');currentProj=null}
function showTab(name,btn){document.querySelectorAll('.tab-panel').forEach(function(p){p.classList.remove('active')});document.querySelectorAll('.modal-tab').forEach(function(b){b.classList.remove('active')});document.getElementById('tab-'+name).classList.add('active');btn.classList.add('active')}

async function saveProject(){if(!currentProj)return;try{await fetch('/api/projects',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:currentProj.id,name:document.getElementById('proj-name').value,description:document.getElementById('proj-desc').value,status:document.getElementById('proj-status').value,notes:document.getElementById('proj-notes').value})});toast(t('saved'));loadProjects()}catch(e){toast('Error','err')}}
async function deleteProject(){if(!currentProj||!confirm(t('confirmDel')+currentProj.name+'?'))return;try{await fetch('/api/projects',{method:'DELETE',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:currentProj.id})});closeModal();loadProjects();toast(t('deleted'))}catch(e){toast('Error','err')}}

async function loadGitInfo(id){try{var r=await fetch('/api/projects/git?id='+id),d=await r.json();document.getElementById('git-branch').textContent=d.branch||'(no git)';document.getElementById('git-branches').textContent=d.branches||'-';document.getElementById('git-status').textContent=d.status||'Clean';document.getElementById('git-log').textContent=d.log||'-';document.getElementById('git-remote').textContent=d.remote||'(no remote)'}catch(e){}}
async function gitAction(action){if(!currentProj)return;var body={id:currentProj.id,action:action};if(action==='commit')body.message=document.getElementById('git-commit-msg').value;if(action==='new_branch'||action==='checkout')body.branch=document.getElementById('git-branch-name').value;if((action==='new_branch'||action==='checkout')&&!body.branch){toast('Branch name required','err');return}try{var r=await fetch('/api/projects/git/action',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});var d=await r.json();toast(action+': OK');loadGitInfo(currentProj.id);if(d.output)document.getElementById('git-status').textContent=d.output}catch(e){toast('Git error: '+e.message,'err')}}

// --- Update System ---
var pendingUpdateInfo=null;
async function checkForUpdates(){
  var btn=document.getElementById('btnCheckUpdate');
  var origHTML=btn.innerHTML;
  btn.disabled=true;btn.innerHTML='<span class="spinner"></span> '+t('checkUpdate');
  var res=document.getElementById('updateResult');
  var info=document.getElementById('updateInfo');
  try{
    var r=await fetch('/api/update/check');
    var d=await r.json();
    if(d.error){res.style.display='block';info.textContent='Error: '+d.error;info.style.borderColor='var(--err)';toast(d.error,'err');return}
    res.style.display='block';
    document.getElementById('currentVerDisplay').textContent=d.current_version;
    if(d.update_available){
      pendingUpdateInfo=d;
      info.innerHTML=t('updateAvail')+'<strong style="color:var(--ok)">'+escPre(d.latest_version)+'</strong>';
      if(d.release_notes){info.innerHTML+='\n\n<strong>'+t('relNotes')+':</strong>\n'+escPre(d.release_notes)}
      if(d.published_at){info.innerHTML+='\n\n'+d.published_at.substring(0,10)}
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
  var btn=document.getElementById('btnApplyUpdate');
  var info=document.getElementById('updateInfo');
  btn.disabled=true;btn.innerHTML='<span class="spinner"></span> '+t('updating');
  info.textContent=t('updating');
  info.style.borderColor='var(--warn)';
  try{
    var r=await fetch('/api/update/apply',{method:'POST'});
    var d=await r.json();
    if(d.success){
      info.textContent=t('updateOk')+'\n\n'+d.message+'\n\n'+d.output;
      info.style.borderColor='var(--ok)';
      toast(t('updateOk'));
      setTimeout(function(){location.reload()},8000)
    }else{
      info.textContent=t('updateFail')+'\n\n'+d.message+'\n\n'+(d.output||'');
      info.style.borderColor='var(--err)';
      toast(t('updateFail'),'err');
      btn.disabled=false;btn.innerHTML=ICN.download+' '+t('applyUpdate')
    }
  }catch(e){info.textContent='Connection lost -- service may be restarting...';info.style.borderColor='var(--warn)';setTimeout(function(){location.reload()},8000)}
}

document.getElementById('chatInput').addEventListener('input',function(){this.style.height='52px';this.style.height=Math.min(this.scrollHeight,120)+'px'});

// Init
loadAppConfig().then(async function(){updateSessionBadge();await loadChatSessions();await loadChatHistory(currentSessionId);await loadModelOptions();await loadSessionSettings();checkOnboarding()});

// --- Session Settings ---
async function loadModelOptions(){
  try{
    var r=await fetch('/api/providers');
    var list=await r.json();
    var sel=document.getElementById('chatModelSelect');
    sel.innerHTML='<option value="">Default (failover)</option>';
    if(list&&list.length){list.forEach(function(p){sel.innerHTML+='<option value="'+escPre(p.id)+'">'+escPre(p.name)+' ('+escPre(p.model)+')</option>'})}
  }catch(e){}
}

async function loadSessionSettings(){
  if(!currentSessionId)return;
  try{
    var r=await fetch('/api/session/settings?session_id='+encodeURIComponent(currentSessionId));
    var s=await r.json();
    if(s.model_id)document.getElementById('chatModelSelect').value=s.model_id;
    if(s.think_level)document.getElementById('chatThinkLevel').value=s.think_level;
    document.getElementById('chatVerbose').checked=!!s.verbose;
  }catch(e){}
}

async function updateSessionSetting(){
  if(!currentSessionId)return;
  var body={
    session_id:currentSessionId,
    model_id:document.getElementById('chatModelSelect').value,
    think_level:document.getElementById('chatThinkLevel').value,
    verbose:document.getElementById('chatVerbose').checked
  };
  try{await fetch('/api/session/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})}catch(e){}
}

async function compactSession(){
  if(!currentSessionId)return;
  if(!confirm('Compact this session? This will summarize the conversation history.'))return;
  try{
    var r=await fetch('/api/chat/compact',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({session_id:currentSessionId})});
    var d=await r.json();
    if(d.error){toast('Compact failed: '+d.error,'err');return}
    toast('Session compacted!');
    await loadChatHistory(currentSessionId);
  }catch(e){toast('Error: '+e.message,'err')}
}

// --- Providers ---
async function loadProviderStatus(){
  try{
    var r=await fetch('/api/providers/status');
    var list=await r.json();
    var c=document.getElementById('providerList');
    if(!list||!list.length){c.innerHTML='<div class="card" style="text-align:center;color:var(--t3)">No providers configured. Set MODELS env var or use the setup wizard.</div>';return}
    c.innerHTML=list.map(function(p){
      var avail=p.available?'<span class="badge badge-on">Available</span>':'<span class="badge badge-paused">Cooldown</span>';
      var fails=p.failures>0?'<span style="color:var(--err);font-size:11px;margin-left:8px">'+p.failures+' failures</span>':'';
      return '<div class="card"><div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px"><div><strong>'+escPre(p.name||p.id)+'</strong> '+avail+fails+'</div><span class="badge badge-active">Priority '+p.priority+'</span></div><div style="font-family:\'JetBrains Mono\',monospace;font-size:12px;color:var(--t3)">Model: '+escPre(p.model)+' - Endpoint: '+escPre(p.endpoint||'')+'</div></div>'
    }).join('');
  }catch(e){}
}

// --- Onboarding Wizard ---
async function checkOnboarding(){
  try{
    var r=await fetch('/api/settings');
    var s=await r.json();
    if(!s.zai_endpoint||s.zai_endpoint===''||s.zai_endpoint==='--------'){
      document.getElementById('onboardModal').classList.add('show');
    }
  }catch(e){}
}
function closeOnboarding(){document.getElementById('onboardModal').classList.remove('show')}
function onbNext(step){
  document.querySelectorAll('.onb-step').forEach(function(s){s.style.display='none'});
  document.getElementById('onb-step-'+step).style.display='block';
}
async function onbValidate(){
  var btn=document.getElementById('onb-validate-btn');
  var res=document.getElementById('onb-validate-result');
  btn.disabled=true;btn.innerHTML='<span class="spinner"></span> Testing...';
  res.style.display='block';res.style.background='var(--bg3)';res.style.borderColor='var(--brd)';res.textContent='Connecting...';
  try{
    var r=await fetch('/api/onboarding/validate',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({
      endpoint:document.getElementById('onb-endpoint').value,
      api_key:document.getElementById('onb-apikey').value,
      model:document.getElementById('onb-model').value
    })});
    var d=await r.json();
    if(d.model_ok){
      res.style.background='rgba(16,185,129,.15)';res.style.color='var(--ok)';
      res.textContent='OK: '+d.message;
    }else if(d.auth_ok){
      res.style.background='rgba(245,158,11,.15)';res.style.color='var(--warn)';
      res.textContent='Warning: '+d.message;
    }else{
      res.style.background='rgba(239,68,68,.15)';res.style.color='var(--err)';
      res.textContent='Error: '+d.message;
    }
  }catch(e){res.style.background='rgba(239,68,68,.15)';res.style.color='var(--err)';res.textContent='Error: '+e.message}
  btn.disabled=false;btn.innerHTML=ICN.search+' Test Connection';
}
async function onbFinish(){
  var settings={
    agent_name:document.getElementById('onb-name').value||'Cronos',
    language:document.getElementById('onb-lang').value||'en',
    zai_endpoint:document.getElementById('onb-endpoint').value,
    zai_api_key:document.getElementById('onb-apikey').value,
    model:document.getElementById('onb-model').value||'gpt-4o',
    workspace_root:document.getElementById('onb-workspace').value||'.',
    telegram_token:document.getElementById('onb-tg-token').value,
    telegram_allow_id:document.getElementById('onb-tg-id').value
  };
  try{
    await fetch('/api/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(settings)});
    appCfg.agent_name=settings.agent_name;appCfg.language=settings.language;applyI18n();
    closeOnboarding();
    toast('Setup complete!');
    await loadSettings();
  }catch(e){toast('Error: '+e.message,'err')}
}
</script>
</body></html>`
}
