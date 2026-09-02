var enabled = true;

chrome.storage.local.get(['server','pass','proxyEnabled','mode'], function(s) {
  document.getElementById('server').value = s.server || '';
  document.getElementById('pass').value = s.pass || '';
  document.getElementById('mode').value = s.mode || 'ai_priority';
  enabled = s.proxyEnabled !== false;
  var srv = s.server || '';
  if (srv) { document.getElementById('sysProxy').textContent = srv; }
  updateUI();
});

function updateUI() {
  document.getElementById('status').textContent = enabled ? '代理已启用' : '代理已关闭';
  document.getElementById('status').className = 'status ' + (enabled ? 'on' : 'off');
  document.getElementById('toggleBtn').textContent = enabled ? '关闭' : '开启';
  document.getElementById('toggleBtn').className = enabled ? 'btn-off' : 'btn-on';
}

document.getElementById('saveBtn').addEventListener('click', function() {
  var server = document.getElementById('server').value.trim();
  var pass = document.getElementById('pass').value.trim();
  var mode = document.getElementById('mode').value;
  if (!server || !pass) { alert('请填写服务器地址和密码'); return; }
  chrome.runtime.sendMessage({ action: 'update', server: server, pass: pass, mode: mode });
  enabled = true;
  updateUI();
});

document.getElementById('toggleBtn').addEventListener('click', function() {
  enabled = !enabled;
  chrome.runtime.sendMessage({ action: enabled ? 'enable' : 'disable' });
  updateUI();
});