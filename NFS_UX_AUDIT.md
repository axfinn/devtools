# NFS 分享模块 UX 优化跟踪

> 2026-06-30 审计产出。共 11 项,按 P0 → P3 排序。
> 优化循环从最高优先级开始,每轮完成一项并提交。

## P0 · 阻塞任务
- [x] #1 访客端首屏骨架屏 + 视频初始化并行 (`NFSShareView.vue:25-34`)
  - [x] (a) 密码弹窗显示文件元数据(文件名 + 大小 + 剩余次数)
  - [x] (b) 视频场景 getUserMedia/fetchRtcConfig/loadQualities 改为 Promise.allSettled 并行(loadQualities 加幂等保护避免重复请求)
  - [x] (c) 转码 overlay 显示清晰度名 + 已生成分片数(后端 /qualities 增 segments 字段,pollTranscoding 每 5s force 刷新)
- [x] #2 视频/文件错误兜底分码 + 重试 CTA (`NFSShareView.vue:485-491,722-734`)
  - 引入 `setError(msg, action?)` + `errorAction` ref,按错误类型挂 CTA(重试 / 复制链接给分享者 / 切换原生模式)
  - 13 处 error 赋值统一走 setError;404/410 → 复制链接;网络/麦克风/视频失败 → 重试;不支持 HLS → 切换原生
- [x] #3 密码弹窗:显示切换 + 不消耗 view 的 `POST /:id/check-password` (`NFSShareView.vue:4-22,495-543`)
  - (a) 显示密码图标(`show-password` prop 已在,本轮确认生效)
  - (b) 后端新增 `POST /:id/check-password`,不消耗 view、不计入日志;前端 confirmPassword 改用专用接口(原本 HEAD /stream 或 / 都会消耗 view)
  - 前端 60s 滑动窗口限速 5 次,UI 实时显示"剩余 X 次尝试机会"
  - (c) 错误后自动聚焦输入框 + 选中已有内容(`focusPasswordInput`)

## P1 · 显著影响
- [x] #4 创建流程合并"显示名称"到高级设置 (`NFSShareTool.vue:114-124,148-150,799-840`)
- [x] #5 大文件下载进度 (`NFSShareView.vue:147-150`)
- [x] #6 统一错误面板加"重新加载"按钮 (`NFSShareView.vue:25-34`)
- [ ] #7 目录搜索/排序 (`NFSShareTool.vue:76-125`)

## P2 · 细节 / a11y
- [ ] #8 a11y 缺口补 aria-label / 键盘焦点环
- [ ] #9 录音开启确认弹窗 + 链接分享后提示气泡

## P3 · 代码层 / 安全
- [ ] #10 超管密码改 cookie 鉴权,不再走 query string (`NFSShareTool.vue` 多处)
- [ ] #11 `confirmPassword` 三分叉清理 + 死代码删除 + `getLogStatus*` map 合并

## 进展
- 2026-06-30 #1a 密码弹窗显示文件名 + 大小 + 剩余查看次数
- 2026-06-30 #1b 视频初始化并行(getUserMedia / fetchRtcConfig / loadQualities),loadQualities 幂等保护
- 2026-06-30 #1c 转码 overlay 显示清晰度 + 已生成分片数(pollTranscoding 每 5s 刷新)
- 2026-06-30 #2 错误兜底分码 + 重试 / 复制 / 切换原生三件套
- 2026-06-30 #3 密码 UX:后端 check-password 不消耗 view + 前端限速 5/min + 错误后自动聚焦
- 2026-06-30 #4 创建表单"显示名称"挪进 el-collapse 高级设置,与录音折叠展示;按钮不再强求 name 必填,留空用 file_path basename 兜底 (40b1e69)
- 2026-06-30 #5 大文件下载进度 (4d34073)
- 2026-06-30 #6 错误面板常驻"重新加载"按钮,errorAction 与 reload 并列布局