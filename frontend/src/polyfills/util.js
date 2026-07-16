// 浏览器端 util stub — readable-stream / debug 只用到 util.debuglog / util.inspect,
// 浏览器里这些功能不需要(都用 console.*),返回空对象让它们 fall back
export default {
  debuglog: () => () => {}, // returns a no-op function
  inspect: (v) => String(v),
  format: (...args) => args.join(' '),
  formatWithOptions: (...args) => args.slice(1).join(' '),
  deprecate: (fn) => fn,
  inherits: (ctor, superCtor) => { ctor.super_ = superCtor },
  promisify: (fn) => (...args) => new Promise((res, rej) => fn(...args, (err, val) => err ? rej(err) : res(val))),
}