# 优化聊天流式输出体验

## Goal

将 QA SSE 的真实接收进度与聊天气泡的视觉输出节奏解耦：网络层持续完整接收 `answer.delta`，显示层通过自适应队列平滑消费，减少 Markdown 重解析和自动滚动抖动，同时不丢失完成、停止或错误时的部分内容。

## Background

- 当前 `apps/web/src/pages/qa/chat/page.tsx:1203` 在每个 `answer.delta` 到达时立即把累计正文写入 Zustand。
- 当前 `apps/web/src/components/chat/chat-messages.tsx:819` 仅在正文超过 2000 字后用一次 RAF 合并到最新快照；短回答仍逐 delta 更新，批量 delta 仍会整块跳出。
- 当前 `apps/web/src/components/chat/chat-messages.tsx:1036` 在每次消息数组变化时同步滚动到底部。
- SSE 解析、事件顺序校验、取消、完成后同批 fatal error 覆盖等传输语义已经存在，本任务不得改变这些协议语义。
- 当前工作区对 `page.tsx` 的未提交改动新增了 `reasoning.delta` 展示；本任务必须保留这些用户改动。

## Requirements

- R1. `answer.delta` 到达后必须立即追加到内存中的完整正文，不得等待全部完成后才开始显示。
- R2. 可见正文必须由基于 `requestAnimationFrame` 的自适应队列消费；短队列保持自然打字节奏，积压增大或网络完成后自动加速追赶。
- R3. Markdown 对应的 React 状态更新频率必须受控，不得严格按单字符或按每个网络 delta 重渲染。
- R4. `answer.completed` 只能在已接收正文安全交付给 UI 后形成最终消息；完成后同一 SSE 批次中的 fatal error 仍可覆盖完成状态。
- R5. 用户停止或 fatal error 必须取消待执行动画，并立即保留截至该时刻已接收的完整部分正文，状态分别保持 `stopped`/`failed`。
- R6. 新请求、组件卸载与动画取消必须清理 RAF，禁止旧队列继续改写新消息。
- R7. 自动滚动必须按 RAF 合并，避免每个网络 delta 同步读写滚动布局。
- R8. 对 `prefers-reduced-motion: reduce` 用户应跳过逐步动画并及时显示完整已接收正文。
- R9. 不修改 SSE 公共事件协议、Gateway OpenAPI、引用标记、报告卡片、附件和侧边栏行为。
- R10. 平滑打字仅应用于最终回答 `answer.delta`；`reasoning.delta`、思考步骤和工具调用保持即时语义，只把同一动画帧中的多次 UI 状态写入合并。

## Acceptance Criteria

- [x] AC1 (R1-R3): 一批包含多个字符或多个 delta 的正文会逐帧平滑出现，且可见状态更新次数显著少于字符数。
- [x] AC2 (R2-R4): 正常完成时队列自适应排空，最终消息内容与所有已接收 delta 严格一致且状态为 `completed`。
- [x] AC3 (R4-R5): 完成后同批 fatal error、主动停止、网络 fatal error 均不会丢失部分正文或被延迟完成回调覆盖。
- [x] AC4 (R6): cancel/unmount 后不再触发可见正文更新或完成回调。
- [x] AC5 (R7): 连续消息更新在同一动画帧最多安排一次自动滚动。
- [x] AC6 (R8): reduced-motion 模式不等待打字动画即可看到完整已接收正文。
- [x] AC7 (R9): 现有聊天消息、思考面板、引用、报告产物和错误/重试测试继续通过。
- [ ] AC8: `bun run --cwd apps/web check`、相关 Vitest、`bun run --cwd apps/web build` 和 `git diff --check` 通过。任务范围 lint/Prettier、50 项相关 Vitest、生产构建和 diff check 已通过；完整 `check` 被 26 个未触及文件的既有 Prettier 基线阻断，完整单测另有报告页并发时序用例偶发超时（单文件 12/12 通过）。
- [x] AC9 (R10): 思考与工具事件不经过打字机排队；同一帧的连续事件合并为一次消息状态提交，最终展示不缺事件。

## Out of Scope

- 修改后端 delta 粒度、SSE 事件格式或重连协议。
- 为打字速度增加用户设置或持久化偏好。
- 改造引用、报告卡片、附件、侧边栏或输入框的产品功能。
