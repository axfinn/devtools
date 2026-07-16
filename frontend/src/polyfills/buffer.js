// 重导出 npm `buffer` 包的 Buffer,同时把它挂到 globalThis.Buffer 上,
// 这样其他模块里裸 `Buffer` 引用(Vite plugin 会替换为 __buf__ 或 globalThis.Buffer)能找到。
//
// 注意:Vite 的 resolve.alias 把 'buffer' 指向本文件,所以这里不能用 'buffer' 字面量,
// 会陷入循环。用相对路径 './buffer-impl'(npm buffer 包没在 node_modules root,
// 实际位置是 node_modules/buffer/index.js,直接通过相对路径绕过 alias)。
import { Buffer, SlowBuffer, INSPECT_MAX_BYTES, kMaxLength } from '../../node_modules/buffer/index.js'

// Vite 的 build 把 'global' 替换成 'windowThis' (它在浏览器里 = globalThis) —
// 但 'globalThis' 不被替换。所以两个名字都要挂,确保 shim 触发的位置能用
// (useScreenRTC.js 是开发时 untransformed,所以用 globalThis;Vite 预构建产物里用 windowThis)
if (typeof globalThis !== 'undefined') {
  globalThis.Buffer = Buffer
  globalThis.SlowBuffer = SlowBuffer
}
if (typeof windowThis !== 'undefined') {
  windowThis.Buffer = Buffer
  windowThis.SlowBuffer = SlowBuffer
}

export { Buffer, SlowBuffer, INSPECT_MAX_BYTES, kMaxLength }
export default { Buffer, SlowBuffer, INSPECT_MAX_BYTES, kMaxLength }