import { ref } from 'vue'
import { ElMessage } from 'element-plus'

/**
 * 统一管理后台登录流程的 composable。
 * 复用之前 7 个 view 一字不差的"输入密码 → 验证 → 缓存 → 自动登录"逻辑。
 *
 * @param {Object} options
 * @param {string} options.storageKey - 凭据存储 key (localStorage/sessionStorage)
 * @param {string} options.verifyEndpoint - 后端验证端点(可用业务端点兼做,如 /api/proxy/status)
 * @param {'body'|'query'|'header'} [options.credentialMode='body'] - 凭据传输方式
 * @param {string} [options.credentialField='password'] - body 或 query 模式下的字段名
 * @param {string} [options.headerName='X-Super-Admin-Password'] - header 模式的 header 名
 * @param {'local'|'session'} [options.storage='local'] - 存储类型
 * @param {boolean} [options.trustStored=false] - true=Hermes/AIChat/AutoDev 模式(信任本地凭据直接进),false=NPS/Proxy 模式(stored 失效主动回滚登录页)
 * @param {(c: object) => boolean} [options.verifyOk] - 自定义"验证通过"判定(用于业务端点兼 verify,如 `{ok:true,status:'ok'}`)
 */
export function useAdminAuth(options) {
  const {
    storageKey,
    verifyEndpoint,
    credentialMode = 'body',
    credentialField = 'password',
    headerName = 'X-Super-Admin-Password',
    storage = 'local',
    trustStored = false,
    verifyOk,
  } = options

  const store = storage === 'session' ? sessionStorage : localStorage
  const passwordInput = ref('')
  const loggingIn = ref(false)
  const isAuthenticated = ref(false)

  const defaultVerifyOk = (data) => data.ok !== false && data.error === undefined

  const verify = async (pwd, silent = false) => {
    const url = verifyEndpoint
    const fetchOpts = { method: 'POST' }
    if (credentialMode === 'body') {
      fetchOpts.headers = { 'Content-Type': 'application/json' }
      fetchOpts.body = JSON.stringify({ [credentialField]: pwd })
    } else if (credentialMode === 'query') {
      // query 模式要 URL-encode 防注入字符
      const u = new URL(url, window.location.origin)
      u.searchParams.set(credentialField, pwd)
      // 走后端"用 super_admin_password 字段名"的 view,Proxy/NPS 历史用 admin_password
      fetchOpts.headers = { 'Content-Type': 'application/json' }
      fetchOpts.body = '{}'
      // 把 query 拼到 url,但 fetchOpts.body 在 query 模式下不再需要
      try {
        const res = await fetch(u.toString(), { method: 'POST' })
        const data = await res.json().catch(() => ({}))
        const ok = res.ok && (verifyOk ? verifyOk(data) : defaultVerifyOk(data))
        if (!ok) throw new Error('auth failed')
        store.setItem(storageKey, pwd)
        isAuthenticated.value = true
        if (!silent) ElMessage.success('登录成功')
        return true
      } catch (e) {
        store.removeItem(storageKey)
        isAuthenticated.value = false
        if (!silent) ElMessage.error('密码错误或已失效')
        return false
      }
    } else if (credentialMode === 'header') {
      fetchOpts.headers = { [headerName]: pwd }
    }
    try {
      const res = await fetch(url, fetchOpts)
      const data = await res.json().catch(() => ({}))
      const ok = res.ok && (verifyOk ? verifyOk(data) : defaultVerifyOk(data))
      if (!ok) throw new Error('auth failed')
      store.setItem(storageKey, pwd)
      isAuthenticated.value = true
      if (!silent) ElMessage.success('登录成功')
      return true
    } catch (e) {
      store.removeItem(storageKey)
      isAuthenticated.value = false
      if (!silent) ElMessage.error('密码错误或已失效')
      return false
    }
  }

  const tryStored = async () => {
    const pwd = store.getItem(storageKey)
    if (!pwd) return false
    if (trustStored) {
      isAuthenticated.value = true
      return true
    }
    // NPS/Proxy 模式:立刻尝试 verify,失败主动回滚
    return await verify(pwd, /* silent */ true)
  }

  const login = async () => {
    if (!passwordInput.value) {
      ElMessage.warning('请输入管理员密码')
      return
    }
    loggingIn.value = true
    await verify(passwordInput.value)
    loggingIn.value = false
  }

  const logout = () => {
    store.removeItem(storageKey)
    isAuthenticated.value = false
    passwordInput.value = ''
    ElMessage.info('已退出登录')
  }

  // 业务请求需要的凭据(给 fetch 调用方用)
  const authQuery = () => ({ [credentialField]: store.getItem(storageKey) || '' })
  const authHeader = (extra = {}) => ({ [headerName]: store.getItem(storageKey) || '', ...extra })
  const authBody = () => ({ [credentialField]: store.getItem(storageKey) || '' })

  return {
    isAuthenticated,
    passwordInput,
    loggingIn,
    login,
    logout,
    tryStored,
    authQuery,
    authHeader,
    authBody,
  }
}
