# Implementation Plan

## 1. Add and verify scheduling primitives

- [x] 新建 `apps/web/src/lib/streaming-text.ts`，实现自适应 `StreamingTextController` 与 `AnimationFrameBatcher`。
- [x] 新建 `apps/web/src/lib/streaming-text.test.ts`，覆盖 feed/finish/cancel/reduced-motion/Unicode 和 batcher 的 merge/flush/cancel。
- [x] 运行该单测，确认 RAF 时序可控。

## 2. Integrate with QA stream lifecycle

- [x] 在 `apps/web/src/pages/qa/chat/page.tsx` 中保留完整正文累加器，只通过 controller 发布可见正文。
- [x] 将 reasoning/tool/citation 的累计 UI patch 接入 frame batcher；保留消息 ID、错误和终态的即时提交。
- [x] 正常完成等待可见队列排空；fatal error/stop 取消动画并立即落完整部分正文。
- [x] 新请求和卸载清理 controller、batcher 与活动流。
- [x] 小心保留当前未提交的 `reasoning.delta`、generation step 和敏感内容清洗改动。

## 3. Simplify rendering and throttle scroll

- [x] 在 `apps/web/src/components/chat/chat-messages.tsx` 移除 2000 字阈值快照节流，保留 memoized Markdown 与 streaming cursor。
- [x] 将自动滚动改为单 RAF 调度并清理旧回调。
- [x] 在 `chat-messages.test.tsx` 补充滚动 RAF 调度/清理覆盖。

## 4. Validation

- [x] `bun run --cwd apps/web test:unit -- src/lib/streaming-text.test.ts src/components/chat/chat-messages.test.tsx src/pages/qa/chat/page.test.ts`
- [ ] `bun run --cwd apps/web check`（typecheck、test typecheck、lint 均通过；完整 format check 被 26 个未触及文件的既有格式基线阻断）
- [x] `bun run --cwd apps/web build`
- [x] `git diff --check`
- [x] 审查 `git diff`，确认没有覆盖工作区既有的 reasoning 展示改动，也没有修改 API/生成文件。

## Risk and Rollback Points

- 最高风险是 `answer.completed`、同批 fatal error 与 controller `onDone` 的竞态；实现后先跑 controller 单测，再检查 page 终态路径。
- frame batcher 使用累计快照而非增量 patch，避免同一帧的工具/推理事件丢失。
- 若集成测试发现完成状态等待过久，优先调整 finishing 消费批量；不退回每 delta Markdown 渲染。
