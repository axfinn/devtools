const DEFAULT_SERVER = "{{.ProxyHost}}";
const DEFAULT_PASS   = "{{.AdminPass}}";

function buildPac(server, mode) {
  // 被墙域名列表（命中则走代理）
  const blocked = [
    'google.com','googleapis.com','googleusercontent.com','gstatic.com','gmail.com',
    'youtube.com','youtu.be','ytimg.com','ggpht.com',
    'twitter.com','x.com','t.co','twimg.com',
    'facebook.com','fbcdn.net','instagram.com','whatsapp.com',
    'telegram.org','t.me',
    'github.com','githubusercontent.com','githubassets.com','ghcr.io',
    'openai.com','chatgpt.com','claude.ai','anthropic.com',
    'notion.so','notionusercontent.com',
    'medium.com','substack.com',
    'reddit.com','redd.it','redditmedia.com','redditstatic.com',
    'wikipedia.org','wikimedia.org',
    'dropbox.com','box.com','onedrive.live.com',
    'spotify.com','netflix.com','twitch.tv',
    'discord.com','discordapp.com','discordapp.net',
    'slack.com','zoom.us',
    'apple.com','icloud.com',
    'amazon.com','amazonaws.com',
    'microsoft.com','live.com','bing.com','msn.com',
    'pixiv.net','fanbox.cc',
    'dl.google.com','storage.googleapis.com',
    'cloudflare.com','cdn.cloudflare.com',
    'jsdelivr.net','unpkg.com','npmjs.com',
    'docker.com','hub.docker.com',
    'stackoverflow.com','stackexchange.com',
    'v2ex.com',
  ];

  // 全部走代理模式
  if (mode === 'global') {
    var blockedStr2 = JSON.stringify(blocked);
    return (
      'function FindProxyForURL(url,host){' +
        'if(isPlainHostName(host)||host==="127.0.0.1"||host==="localhost"||' +
          'isInNet(host,"10.0.0.0","255.0.0.0")||isInNet(host,"172.16.0.0","255.240.0.0")||' +
          'isInNet(host,"192.168.0.0","255.255.0.0"))return "DIRECT";' +
        'return "PROXY ' + server + '";' +
      '}'
    );
  }

  // 智能分流模式（默认）：
  // 被墙域名 → PROXY（代理不通再直连）
  // 其余 → DIRECT; PROXY（直连不通再走代理）
  var blockedStr = JSON.stringify(blocked);
  return (
    'var BLOCKED=' + blockedStr + ';' +
    'function FindProxyForURL(url,host){' +
      'if(isPlainHostName(host)||host==="127.0.0.1"||host==="localhost"||' +
        'isInNet(host,"10.0.0.0","255.0.0.0")||isInNet(host,"172.16.0.0","255.240.0.0")||' +
        'isInNet(host,"192.168.0.0","255.255.0.0"))return "DIRECT";' +
      'for(var i=0;i<BLOCKED.length;i++){var d=BLOCKED[i];if(host===d||host.slice(-(d.length+1))==="."+d)return "PROXY ' + server + '; DIRECT";}' +
      'return "DIRECT; PROXY ' + server + '";' +
    '}'
  );
}

function applyProxy(server, mode) {
  const pac = buildPac(server, mode || 'ai_priority');
  chrome.proxy.settings.set({
    value: { mode: 'pac_script', pacScript: { data: pac } },
    scope: 'regular'
  });
}

function disableProxy() {
  chrome.proxy.settings.clear({ scope: 'regular' });
}

function loadAndApply() {
  chrome.storage.local.get(['proxyEnabled', 'server', 'mode'], (s) => {
    if (s.proxyEnabled !== false) {
      applyProxy(s.server || DEFAULT_SERVER, s.mode || 'ai_priority');
    } else {
      disableProxy();
    }
  });
}

// 启动时立即用默认值同步设置代理，避免 storage 异步导致新标签页第一个请求走直连
applyProxy(DEFAULT_SERVER, 'ai_priority');

chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.set({ server: DEFAULT_SERVER, pass: DEFAULT_PASS, proxyEnabled: true, mode: 'ai_priority' });
  loadAndApply();
});
chrome.runtime.onStartup.addListener(loadAndApply);
loadAndApply();

// MV3 service worker 会被 Chrome 休眠，用 alarm 每25秒唤醒一次保持代理设置
chrome.alarms.create('keepAlive', { periodInMinutes: 0.4 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'keepAlive') loadAndApply();
});

// 自动填充代理认证
chrome.webRequest.onAuthRequired.addListener(
  (details) => {
    if (!details.isProxy) return {};
    return new Promise((resolve) => {
      chrome.storage.local.get(['pass'], (s) => {
        resolve({ authCredentials: { username: 'proxy', password: s.pass || DEFAULT_PASS } });
      });
    });
  },
  { urls: ['<all_urls>'] }
);

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.action === 'update') {
    chrome.storage.local.set({ server: msg.server, pass: msg.pass, mode: msg.mode, proxyEnabled: true }, () => {
      applyProxy(msg.server, msg.mode);
    });
  } else if (msg.action === 'disable') {
    chrome.storage.local.set({ proxyEnabled: false });
    disableProxy();
  } else if (msg.action === 'enable') {
    chrome.storage.local.get(['server', 'mode'], (s) => {
      chrome.storage.local.set({ proxyEnabled: true });
      applyProxy(s.server || DEFAULT_SERVER, s.mode || 'ai_priority');
    });
  }
  sendResponse({});
  return true;
});