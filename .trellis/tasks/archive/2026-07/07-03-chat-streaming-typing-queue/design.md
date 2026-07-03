# Design: 聊天流式打字队列

## Architecture

流式链路分为两个彼此独立的进度：

```text
SSE answer.delta
  ├─ content accumulator：立即保存完整已接收正文（网络事实）
  └─ StreamingTextController：按 RAF 自适应消费（可见正文）
       └─ Zustand assistant message.content
            └─ ChatMessages Markdown + RAF auto-scroll
```

`page.tsx` 继续拥有一次回答的 SSE 生命周期和完整累加器。新增的浏览器侧 controller 只负责视觉节奏，不进入 API parser，也不改变 Zustand 的持久结构。

思考、推理、工具、引用等结构化事件不进入打字机。它们的累计数据仍立即写入函数内变量，但 UI patch 通过一个浅合并的 RAF 调度器提交；一帧内后到的累计快照覆盖前一个快照，从而保留最终语义并减少重复 Zustand 更新。

## Components and Boundaries

### `apps/web/src/lib/streaming-text.ts`

- 提供 `StreamingTextController`。
- 输入为增量字符串，内部按 grapheme（有 `Intl.Segmenter` 时）排队。
- 默认约 30 FPS，短队列少量消费，中长队列逐级加速，`finish()` 后在有限帧内排空。
- `cancel()` 必须取消 RAF，且之后不得再调用 `onUpdate`/`onDone`。
- reduced-motion 模式每帧合并并消费全部积压，不做逐字动画。
- 提供小型 `AnimationFrameBatcher<T>`，用于把同一帧内的结构化 UI patch 合并为一次提交，并支持 `flush()`/`cancel()`。

### `apps/web/src/pages/qa/chat/page.tsx`

- `content` 保持网络层完整答案累加器。
- `onAnswerDelta` 只追加 `content` 并 `feed(delta)`；controller 的 `onUpdate` 才更新 assistant 的可见 `content`。
- `onAnswerCompleted` 保存最终消息 ID、最终思考/引用/产物快照并调用 `finish()`；controller 排空后才提交最终完整正文和 `completed`。
- fatal error 和主动停止先取消 controller/batcher，再用完整 `content` 立即提交 `failed`/`stopped`，防止排队正文丢失或延迟完成覆盖错误。
- 新流开始和页面卸载时取消旧 controller/batcher；卸载仍终止活动 SSE。
- 思考、工具与引用 handler 使用 frame batcher；ID 捕获、最终状态与错误状态仍即时提交。

### `apps/web/src/components/chat/chat-messages.tsx`

- 删除现有“仅超过 2000 字才同步最新快照”的二次节流；正文输入已经由 controller 控速，组件只 memoize Markdown。
- 自动滚动改为 RAF 调度并在 effect cleanup 中取消，避免连续消息更新导致同步 layout 读写。
- 保留 cursor、elapsed、ThinkPanel、citation 和 artifact 的现有渲染契约。

## State and Event Semantics

### Normal completion

1. `answer.delta` 立即累加到 `content`，可见正文逐帧追赶。
2. `answer.completed` 不立即把完整正文整块刷出，也不立即清除全局 streaming。
3. controller 切换到 finishing，加速但非瞬间排空。
4. `onDone` 提交 `content`、最终元数据和 `completed`；用 microtask 清除 streaming，以保留同批 fatal error 的覆盖机会。

### Fatal error

1. 取消 controller 与待提交的 frame patch。
2. 用当前完整 `content` 和累计元数据提交 `failed`。
3. 后续 abort callback 识别消息已终态，不再覆盖。

### User stop / unmount

- 主动停止：取消动画并立即显示完整部分正文，状态为 `stopped`。
- 页面卸载：取消动画和待提交帧，并 abort SSE；不允许旧 RAF 在卸载后继续写 store。

### Reduced motion

- 不使用逐 grapheme 的视觉节奏；每个 RAF 把当前积压一次显示，仍保留帧级合并与完整内容保证。

## Compatibility and Rollback

- 不改 API、OpenAPI、SSE parser、消息 DTO 或 Zustand 持久化结构。
- 仅新增前端运行时 helper，旧浏览器若无 `Intl.Segmenter` 则回退到 `Array.from` 的 code-point 切分。
- 回滚时可移除 controller/batcher 集成并恢复 `onAnswerDelta -> patchAssistant({ content })`；无数据迁移。
- 本次属于交互性能优化，不提升前端版本号。

## Test Strategy

- controller 单元测试：逐帧消费、自适应追赶、完成排空、取消、reduced-motion、Unicode。
- batcher 单元测试：同帧合并、flush、cancel。
- ChatMessages 组件测试：连续消息更新只保留一个待执行滚动 RAF，卸载会取消。
- 现有 chat message、page helper、chat API 测试回归。
