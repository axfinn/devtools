# NFS 分享模块 UX 优化跟踪

> 2026-06-30 审计产出。共 11 项,按 P0 → P3 排序。
> 优化循环从最高优先级开始,每轮完成一项并提交。

## P0 · 阻塞任务
- [ ] #1 访客端首屏骨架屏 + 视频初始化并行 (`NFSShareView.vue:25-34`)
- [ ] #2 视频/文件错误兜底分码 + 重试 CTA (`NFSShareView.vue:485-491,722-734`)
- [ ] #3 密码弹窗:显示切换 + 不消耗 view 的 `POST /:id/check-password` (`NFSShareView.vue:4-22,495-543`)

## P1 · 显著影响
- [ ] #4 创建流程合并"显示名称"到高级设置 (`NFSShareTool.vue:114-124,148-150,799-840`)
- [ ] #5 大文件下载进度 (`NFSShareView.vue:147-150`)
- [ ] #6 统一错误面板加"重新加载"按钮 (`NFSShareView.vue:25-34`)
- [ ] #7 目录搜索/排序 (`NFSShareTool.vue:76-125`)

## P2 · 细节 / a11y
- [ ] #8 a11y 缺口补 aria-label / 键盘焦点环
- [ ] #9 录音开启确认弹窗 + 链接分享后提示气泡

## P3 · 代码层 / 安全
- [ ] #10 超管密码改 cookie 鉴权,不再走 query string (`NFSShareTool.vue` 多处)
- [ ] #11 `confirmPassword` 三分叉清理 + 死代码删除 + `getLogStatus*` map 合并

## 进展
- (待开始)