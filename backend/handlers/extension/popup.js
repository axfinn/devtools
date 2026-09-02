// 所有 UI 状态以 chrome.storage 为单一可信源,本地 enabled 不缓存:
// popup 打开时读 storage,操作后再次读 storage 同步,
// 避免 background.js 异步写未完成时 UI 提前翻面造成状态错觉。
function refreshFromStorage(cb) {
  chrome.storage.local.get(['server','pass','proxyEnabled','mode'], function(s) {
    document.getElementById('server').value = s.server || '';
    document.getElementById('pass').value = s.pass || '';
    document.getElementById('mode').value = s.mode || 'ai_priority';
    var srv = s.server || '';
    if (srv) { document.getElementById('sysProxy').textContent = srv; }
    var enabled = s.proxyEnabled !== false;
    document.getElementById('status').textContent = enabled ? '代理已启用' : '代理已关闭';
    document.getElementById('status').className = 'status ' + (enabled ? 'on' : 'off');
    document.getElementById('toggleBtn').textContent = enabled ? '关闭' : '开启';
    document.getElementById('toggleBtn').className = enabled ? 'btn-off' : 'btn-on';
    if (cb) cb(enabled);
  });
}

refreshFromStorage();

document.getElementById('saveBtn').addEventListener('click', function() {
  var server = document.getElementById('server').value.trim();
  var pass = document.getElementById('pass').value.trim();
  var mode = document.getElementById('mode').value;
  if (!server || !pass) { alert('请填写服务器地址和密码'); return; }
  chrome.runtime.sendMessage({ action: 'update', server: server, pass: pass, mode: mode });
  // 等 background.js 写完 storage 再 refresh,防止 UI 提前翻面
  refreshFromStorage();
});

document.getElementById('toggleBtn').addEventListener('click', function() {
  // 先取当前真实状态再翻转,避免本地状态与 storage 漂移
  chrome.storage.local.get(['proxyEnabled'], function(s) {
    var wasEnabled = s.proxyEnabled !== false;
    chrome.runtime.sendMessage({ action: wasEnabled ? 'disable' : 'enable' });
    refreshFromStorage();
  });
});
