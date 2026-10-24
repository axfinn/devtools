/*
 * pet-widget.js —— 魔法宠物（云帆 Yunfan）渲染引擎
 * ---------------------------------------------------------------------------
 * 替换原 1.9MB 黑盒打包产物（React + Three 混编），用干净的 Three.js Web Component 实现。
 *
 * 复用（不改动）：
 *   - GlobalPet.vue / PetTool.vue / MiniMaxStudioTool.vue 外壳与调用方式
 *   - petWidgetLoader.js 的加载逻辑（依旧 <script src="/widgets/pet-widget.js">）
 *   - pet-themes.js 的主题色（THEMES 与之一致）
 *   - /widgets/voices/*.mp3 预录语音（动作/欢呼/打招呼/变身全部复用）
 *
 * 兼容原 <pet-widget> 契约：
 *   属性: theme / form / width / height / voice / no-shell
 *   方法: setForm(id) / setTheme(id) / speak('cheer'|'greeting'|'bye'|文本)
 */
(function () {
  'use strict';

  // 资源路径（相对于站点根，与 petWidgetLoader 的 /widgets 约定一致）
  var MODEL_URL = '/widgets/models/yunfan.glb';
  var VOICE_BASE = '/widgets/voices/';
  var THREE_URL = '/widgets/vendor/three.module.js';
  var GLTF_URL = '/widgets/vendor/jsm/loaders/GLTFLoader.js';

  // 主题（与 src/constants/pet-themes.js 保持一致）
  var THEMES = {
    flame: { name: '烈焰', accent: '#ff7a4d' },
    tide: { name: '深海', accent: '#5cc8ff' },
    forest: { name: '森语', accent: '#7dffaf' },
    cosmic: { name: '星海', accent: '#b88dff' },
    prism: { name: '棱镜', accent: '#5cf2ff' }
  };

  // 云帆自带动作（按模型实际 clip 名过滤，缺失则回退到全部）
  var FRIENDLY = ['Wave', 'Yes', 'No', 'Dance', 'Jump', 'ThumbsUp', 'Punch', 'WalkJump'];

  // 动作 → 复用语音映射
  var ACTION_VOICE = {
    Wave: 'act_wave.mp3',
    Dance: 'act_dance.mp3',
    Jump: 'act_jump.mp3',
    ThumbsUp: 'act_attack.mp3',
    Punch: 'act_attack.mp3',
    Yes: 'idle_01.mp3'
  };
  var FORM_VOICE = {
    default: 'form_default.mp3',
    cat: 'form_cat.mp3',
    robot: 'form_robot.mp3',
    ghost: 'form_ghost.mp3',
    star: 'form_star.mp3'
  };
  var FORM_TINT = {
    default: 0xffffff,
    cat: 0xffd9a0,
    robot: 0xbfe6ff,
    ghost: 0xd9d2ff,
    star: 0xfff2a0
  };

  var THREE = null;
  var GLTFLoader = null;

  // ---------------------------------------------------------------------------
  // 全局语音总开关（所有 <pet-widget> 实例共享，刷新后仍然生效）
  // ---------------------------------------------------------------------------
  // 沿用合并前 main 的同一个持久化键；缺失 = 开启（不改变首次访问体验）。
  var VOICE_ENABLED_KEY = 'pet-widget.enabled';
  // 全局开关变化时广播给页面上的 UI（让 🔊/🔇 图标与实际发声状态保持一致）
  var VOICE_CHANGE_EVENT = 'pet-voice-change';

  function readGlobalVoice() {
    try {
      var v = localStorage.getItem(VOICE_ENABLED_KEY);
      return v === null ? true : v === '1';
    } catch (e) { return true; } // 隐私模式 / 无 localStorage ⇒ 降级为默认开启
  }

  var globalVoiceOn = readGlobalVoice();

  function emitVoiceChange() {
    try {
      document.dispatchEvent(new CustomEvent(VOICE_CHANGE_EVENT, { detail: { enabled: globalVoiceOn } }));
    } catch (e) { /* ignore */ }
  }

  // 写回全局开关（持久化）；值变化时广播，避免 UI 图标与实际状态脱节
  function setGlobalVoice(on) {
    var next = !!on;
    var changed = next !== globalVoiceOn;
    globalVoiceOn = next;
    try { localStorage.setItem(VOICE_ENABLED_KEY, globalVoiceOn ? '1' : '0'); } catch (e) { /* ignore */ }
    if (changed) emitVoiceChange();
  }

  // 发声前刷新：页面 UI（或另一个标签页）改了存储，这里立刻跟上
  function refreshGlobalVoice() {
    var next = readGlobalVoice();
    if (next !== globalVoiceOn) {
      globalVoiceOn = next;
      emitVoiceChange();
    }
    return globalVoiceOn;
  }

  // 唯一的 MP3 出口：allowed 为假时连 Audio 都不创建（不是"播了但静音"）
  function playVoice(file, volume, allowed) {
    if (!allowed) return;
    try {
      var a = new Audio(VOICE_BASE + file);
      a.volume = volume == null ? 0.6 : volume;
      var p = a.play();
      if (p && p.catch) p.catch(function () {});
    } catch (e) { /* 自动播放可能被拦截，忽略 */ }
  }

  // 主题预录语音：theme_<id>.mp3（id 取自 THEMES，未知主题不发声、也绝不回退成文本朗读）
  function playThemeVoice(themeId, allowed) {
    if (THEMES[themeId]) playVoice('theme_' + themeId + '.mp3', undefined, allowed);
  }

  function pad2(n) { return n < 10 ? '0' + n : '' + n; }

  class PetWidgetElement extends HTMLElement {
    constructor() {
      super();
      this._disposed = false;
      this._initStarted = false;
      this._ready = false;
      this._raf = 0;
      this._randTimer = null;
      this._actions = {};
      this._mixer = null;
      this._current = null;
      this._model = null;
      this._baseY = 0;
      this._bob = 0;
      this._accent = THEMES.prism.accent;
      this._form = 'default';
      this._voiceOn = false;
      this._clock = null;
    }

    static get observedAttributes() { return ['theme', 'voice', 'form', 'width', 'height']; }

    connectedCallback() {
      if (this._initStarted) return;
      this._initStarted = true;
      this._init();
    }

    disconnectedCallback() { this._dispose(); }

    attributeChangedCallback(name, _old, val) {
      if (name === 'theme') this._applyTheme(val);
      else if (name === 'voice') {
        this._voiceOn = this.hasAttribute('voice');
        // 元素已连接后 voice 属性的变化 = 用户意图 ⇒ 写回全局开关（刷新/换页后仍生效）。
        // 首次挂载（尚未连接）时只决定元素自身开关，绝不覆盖持久化的全局开关。
        if (this.isConnected) setGlobalVoice(this._voiceOn);
      } else if (name === 'form') this.setForm(val);
      else if (name === 'width' || name === 'height') this._resize();
    }

    // 有效发声 = 全局开关 && 元素自身开关（voice 属性）
    _voiceAllowed() {
      return refreshGlobalVoice() && this._voiceOn;
    }

    _init() {
      var self = this;
      // 让宿主填满父容器
      self.style.display = 'block';
      self.style.width = '100%';
      self.style.height = '100%';
      self.style.position = 'relative';

      var canvas = document.createElement('canvas');
      canvas.style.cssText = 'display:block;width:100%;height:100%;';
      self.appendChild(canvas);
      self._canvas = canvas;

      self._voiceOn = self.hasAttribute('voice');
      self._applyTheme(self.getAttribute('theme'));
      if (self.getAttribute('form')) self._form = self.getAttribute('form');

      Promise.resolve()
        .then(function () {
          if (THREE) return;
          return import(THREE_URL).then(function (mod) {
            THREE = mod;
            return import(GLTF_URL).then(function (g) { GLTFLoader = g.GLTFLoader; });
          });
        })
        .then(function () {
          if (self._disposed) return;
          self._setupScene(canvas);
          self._loadModel();
          self._ready = true;
        })
        .catch(function (e) {
          console.error('[pet-widget] 初始化失败（three/模型加载异常）', e);
        });
    }

    _setupScene(canvas) {
      var self = this;
      var renderer = new THREE.WebGLRenderer({ canvas: canvas, antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.outputColorSpace = THREE.SRGBColorSpace;
      renderer.shadowMap.enabled = true;
      renderer.shadowMap.type = THREE.PCFSoftShadowMap;
      self._renderer = renderer;

      var scene = new THREE.Scene();
      self._scene = scene;

      var camera = new THREE.PerspectiveCamera(35, 1, 0.1, 100);
      camera.position.set(0, 1.0, 3.4);
      camera.lookAt(0, 0.85, 0);
      self._camera = camera;

      scene.add(new THREE.HemisphereLight(0xffffff, 0x202830, 1.1));
      var key = new THREE.DirectionalLight(0xffffff, 2.0);
      key.position.set(2, 5, 3);
      key.castShadow = true;
      key.shadow.mapSize.set(1024, 1024);
      key.shadow.camera.near = 1;
      key.shadow.camera.far = 20;
      scene.add(key);
      var fill = new THREE.DirectionalLight(0x88aaff, 0.5);
      fill.position.set(-3, 2, -2);
      scene.add(fill);

      // 地面：ShadowMaterial 只渲染阴影、其余透明 —— 适配透明浮动宠物窗
      var ground = new THREE.Mesh(
        new THREE.CircleGeometry(2.2, 48),
        new THREE.ShadowMaterial({ opacity: 0.28 })
      );
      ground.rotation.x = -Math.PI / 2;
      ground.receiveShadow = true;
      scene.add(ground);
      self._ground = ground;

      // 主题补光：低强度、偏后上方的边缘光，只添氛围、不夺走云帆本色
      var accent = new THREE.PointLight(new THREE.Color(self._accent), 0.35, 10);
      accent.position.set(1.6, 2.4, -1.6);
      scene.add(accent);
      self._accentLight = accent;

      self._clock = new THREE.Clock();
      self._resize();
      if (typeof ResizeObserver !== 'undefined') {
        self._ro = new ResizeObserver(function () { self._resize(); });
        self._ro.observe(self);
      }
      canvas.addEventListener('pointerdown', function () { self._onTap(); });

      self._loop();
    }

    _resize() {
      var w = parseInt(this.getAttribute('width'), 10) || this.clientWidth || 280;
      var h = parseInt(this.getAttribute('height'), 10) || this.clientHeight || 300;
      if (this._renderer) {
        this._renderer.setSize(w, h, false);
        this._camera.aspect = w / h;
        this._camera.updateProjectionMatrix();
      }
    }

    _applyTheme(id) {
      var t = THEMES[id] || THEMES.prism;
      this._accent = t.accent;
      if (this._accentLight) this._accentLight.color = new THREE.Color(t.accent);
    }

    _loadModel() {
      var self = this;
      var loader = new GLTFLoader();
      loader.load(
        MODEL_URL,
        function (gltf) {
          if (self._disposed) return;
          var model = gltf.scene;
          model.traverse(function (o) {
            if (o.isMesh) {
              o.castShadow = true; o.receiveShadow = true;
              // 记录材质原色，供 setForm 从原色轻度混合（不覆盖云帆本色）
              if (o.material) {
                var ms = Array.isArray(o.material) ? o.material : [o.material];
                ms.forEach(function (m) { if (m.color && !m.__origColor) m.__origColor = m.color.clone(); });
              }
            }
          });

          // 自动居中 + 缩放（仅合并可见 Mesh 的包围盒，避免骨骼/空节点污染）
          var box = new THREE.Box3();
          var tmp = new THREE.Box3();
          model.updateMatrixWorld(true);
          model.traverse(function (o) {
            if (o.isMesh && o.geometry) {
              o.geometry.computeBoundingBox();
              if (o.geometry.boundingBox) {
                tmp.copy(o.geometry.boundingBox).applyMatrix4(o.matrixWorld);
                box.union(tmp);
              }
            }
          });
          var size = box.getSize(new THREE.Vector3());
          var s = 1.6 / (size.y || 1);
          model.scale.setScalar(s);
          model.updateMatrixWorld(true);

          box.makeEmpty();
          model.traverse(function (o) {
            if (o.isMesh && o.geometry) {
              o.geometry.computeBoundingBox();
              if (o.geometry.boundingBox) {
                tmp.copy(o.geometry.boundingBox).applyMatrix4(o.matrixWorld);
                box.union(tmp);
              }
            }
          });
          var center = box.getCenter(new THREE.Vector3());
          model.position.x -= center.x;
          model.position.z -= center.z;
          model.position.y -= box.min.y;
          self._scene.add(model);
          self._model = model;
          self._baseY = model.position.y;

          // 动画
          self._mixer = new THREE.AnimationMixer(model);
          self._actions = {};
          gltf.animations.forEach(function (clip) {
            self._actions[clip.name] = self._mixer.clipAction(clip);
          });
          self._friendly = FRIENDLY.filter(function (n) { return self._actions[n]; });
          if (!self._friendly.length) self._friendly = Object.keys(self._actions);

          self._play('Idle', 0.4);
          self._scheduleRandom();
          self.setForm(self._form); // 应用当前形态着色
          setTimeout(function () { if (!self._disposed) self.speak('greeting'); }, 600);
        },
        undefined,
        function (e) { console.error('[pet-widget] 模型加载失败', e); }
      );
    }

    _play(name, dur) {
      var a = this._actions[name];
      if (!a || this._current === a) return;
      a.reset().fadeIn(dur == null ? 0.3 : dur).play();
      if (this._current) this._current.fadeOut(0.3);
      this._current = a;
    }

    _scheduleRandom() {
      var self = this;
      clearTimeout(self._randTimer);
      self._randTimer = setTimeout(function () {
        if (!self._friendly || !self._friendly.length) return;
        var name = self._friendly[Math.floor(Math.random() * self._friendly.length)];
        self._play(name, 0.3);
        if (ACTION_VOICE[name]) playVoice(ACTION_VOICE[name], undefined, self._voiceAllowed());
        var clip = self._actions[name];
        var d = (clip ? clip.getClip().duration * 1000 : 1200) + 300;
        setTimeout(function () {
          if (self._current === self._actions[name]) self._play('Idle', 0.5);
        }, d);
        self._scheduleRandom();
      }, 4000 + Math.random() * 5000);
    }

    _onTap() {
      if (!this._friendly || !this._friendly.length) return;
      var name = this._friendly[Math.floor(Math.random() * this._friendly.length)];
      this._play(name, 0.2);
      if (ACTION_VOICE[name]) playVoice(ACTION_VOICE[name], undefined, this._voiceAllowed());
    }

    setForm(id) {
      this._form = id || 'default';
      if (this._model) {
        var tintHex = FORM_TINT[this._form] || 0xffffff;
        // default 形态还原本色；其它形态在原色上轻度混合（0.3），保留云帆辨识度
        var factor = this._form === 'default' ? 0 : 0.3;
        var tc = new THREE.Color(tintHex);
        this._model.traverse(function (o) {
          if (o.isMesh && o.material) {
            var mats = Array.isArray(o.material) ? o.material : [o.material];
            mats.forEach(function (m) {
              if (!m.color || !m.__origColor) return;
              if (factor === 0) m.color.copy(m.__origColor);
              else m.color.copy(m.__origColor).lerp(tc, factor);
            });
          }
        });
      }
      if (FORM_VOICE[this._form]) playVoice(FORM_VOICE[this._form], 0.5, this._voiceAllowed());
    }

    setTheme(id) {
      if (!id) return;
      this._applyTheme(id);
      if (this.getAttribute('theme') !== id) this.setAttribute('theme', id);
      // 换色后播一次该主题的预录欢迎语（对齐 main「主题切换也播一次」）
      playThemeVoice(id, this._voiceAllowed());
    }

    speak(kind) {
      var self = this;
      var allowed = this._voiceAllowed();
      if (kind === 'cheer') {
        playVoice('cheer_' + pad2(Math.floor(Math.random() * 19) + 1) + '.mp3', undefined, allowed);
        self._play('ThumbsUp', 0.2);
        if (allowed && 'speechSynthesis' in window) {
          var u = new SpeechSynthesisUtterance('你已经很棒啦！');
          u.lang = 'zh-CN'; u.rate = 1;
          window.speechSynthesis.cancel(); window.speechSynthesis.speak(u);
        }
        return;
      }
      if (kind === 'greeting') { playVoice('greeting_hi.mp3', undefined, allowed); self._play('Wave', 0.2); return; }
      if (kind === 'bye') { playVoice('greeting_bye.mp3', undefined, allowed); return; }
      // 主题欢迎语：'theme_<id>' 播放预录主题语音，绝不把 key 当文本念出来
      if (typeof kind === 'string' && kind.indexOf('theme_') === 0) {
        playThemeVoice(kind.slice('theme_'.length), allowed);
        return;
      }
      if (allowed && 'speechSynthesis' in window && typeof kind === 'string') {
        var u2 = new SpeechSynthesisUtterance(kind);
        u2.lang = 'zh-CN'; u2.rate = 1;
        window.speechSynthesis.cancel(); window.speechSynthesis.speak(u2);
      }
    }

    _loop() {
      if (this._disposed) return;
      var self = this;
      self._raf = requestAnimationFrame(function () { self._loop(); });
      var dt = self._clock ? self._clock.getDelta() : 0;
      if (self._mixer) self._mixer.update(dt);
      if (self._model && self._baseY != null) {
        self._bob += dt;
        self._model.position.y = self._baseY + Math.sin(self._bob * 1.5) * 0.02;
        self._model.rotation.y = Math.sin(self._bob * 0.5) * 0.15;
      }
      if (self._renderer && self._scene && self._camera) {
        self._renderer.render(self._scene, self._camera);
      }
    }

    _dispose() {
      this._disposed = true;
      cancelAnimationFrame(this._raf);
      clearTimeout(this._randTimer);
      if (this._ro) this._ro.disconnect();
      if (this._scene) {
        this._scene.traverse(function (o) {
          if (o.geometry) o.geometry.dispose();
          if (o.material) {
            var ms = Array.isArray(o.material) ? o.material : [o.material];
            ms.forEach(function (m) { m.dispose(); });
          }
        });
      }
      if (this._renderer) this._renderer.dispose();
    }
  }

  if (typeof customElements !== 'undefined' && !customElements.get('pet-widget')) {
    customElements.define('pet-widget', PetWidgetElement);
  }
})();
