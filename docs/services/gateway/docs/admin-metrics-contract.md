# 管理后台总览与统计聚合接口文档

本文定义 Gateway 面向管理后台的跨服务聚合接口。`GET /api/v1/admin/overview`
和 `GET /api/v1/admin/metrics` 已是 Gateway active contract；当前
`services/gateway` route matrix 仍返回稳定 `501 not_implemented`，本文件只补齐
接口口径和后续实现验收边界，不代表本 PR 已完成后端聚合或前端页面改造。

## 状态与边界

| 项目 | 说明 |
| --- | --- |
| 契约状态 | active contract，机器可读 schema 以 `docs/services/gateway/api/public.openapi.yaml` 为准。 |
| 当前实现 | Gateway 已注册 route，但仍按稳定 `not_implemented` 占位返回。 |
| 数据归属 | Gateway 只聚合；Auth、Knowledge、Document 和 QA 分别提供自己的权威事实或统计接口。 |
| 响应 envelope | JSON 成功响应统一为 `{ "data": ..., "requestId": "..." }`，错误响应统一为 `{ "error": ... }`。 |
| 管理权限 | 需要 `bearerAuth`，并限制为 `admin`、`super_admin` 或等价管理权限。 |

## GET /api/v1/admin/overview

返回管理后台卡片所需的当前快照。该接口适合页面首次加载和手动刷新，不作为实时运维监控。

### 响应字段

| 字段 | 类型 | 口径 |
| --- | --- | --- |
| `totals.userCount` | integer | Auth 拥有的未软删除用户总数，包含 `active`、`disabled` 和 `locked` 状态。 |
| `totals.knowledgeBaseCount` | integer | Knowledge 拥有的未删除知识库总数。 |
| `totals.documentCount` | integer | Knowledge 拥有的未删除知识库文档总数。 |
| `totals.chunkCount` | integer | Knowledge runtime 可被公开检索或展示的切片总数，不包含已删除文档的内部切片。 |
| `totals.reportTemplateCount` | integer | Document 拥有的未软删除报告模板总数。 |
| `totals.reportRecordCount` | integer | Document 拥有的未软删除报告记录总数。 |
| `totals.qaCount` | integer | QA 拥有的用户提问总次数，默认以未删除会话中的 `messages.role = user` 为事实来源。 |
| `updatedAt` | date-time | Gateway 完成本次聚合的时间。 |

### 成功示例

```json
{
  "data": {
    "totals": {
      "userCount": 42,
      "knowledgeBaseCount": 12,
      "documentCount": 128,
      "chunkCount": 9800,
      "reportTemplateCount": 7,
      "reportRecordCount": 36,
      "qaCount": 512
    },
    "updatedAt": "2026-07-04T08:30:00Z"
  },
  "requestId": "req_admin_overview_001"
}
```

## GET /api/v1/admin/metrics

返回管理后台近 30 天趋势图所需的时间序列。默认查询 `days=30`、
`granularity=daily`；`days` 可在 `1..90` 范围内调整。

### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `days` | integer | `30` | 回看天数，取值范围 `1..90`，包含当天。 |
| `granularity` | string | `daily` | 时间桶粒度；当前契约保留 `daily` 和 `hourly`，管理后台默认使用 `daily`。 |

### 响应字段

`series` 下的字段与总览卡片一一对应。趋势点的 `count` 表示该时间桶内新增或发生的次数，不表示历史累计快照。

| 字段 | 口径 |
| --- | --- |
| `series.userCount` | 桶内新增用户数。 |
| `series.knowledgeBaseCount` | 桶内新增知识库数。 |
| `series.documentCount` | 桶内新增知识库文档数。 |
| `series.chunkCount` | 桶内新增或变为可用的切片数。 |
| `series.reportTemplateCount` | 桶内新增报告模板数。 |
| `series.reportRecordCount` | 桶内新增报告记录数。 |
| `series.qaCount` | 桶内用户提问次数。 |

每个趋势点包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `date` | date-time | 时间桶起点。`daily` 粒度使用 UTC 当日 `00:00:00Z`。 |
| `count` | integer | 该桶内的非负计数。 |

### 成功示例

```json
{
  "data": {
    "days": 30,
    "granularity": "daily",
    "series": {
      "userCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 2 },
        { "date": "2026-07-04T00:00:00Z", "count": 1 }
      ],
      "knowledgeBaseCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 1 },
        { "date": "2026-07-04T00:00:00Z", "count": 0 }
      ],
      "documentCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 5 },
        { "date": "2026-07-04T00:00:00Z", "count": 3 }
      ],
      "chunkCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 420 },
        { "date": "2026-07-04T00:00:00Z", "count": 260 }
      ],
      "reportTemplateCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 0 },
        { "date": "2026-07-04T00:00:00Z", "count": 1 }
      ],
      "reportRecordCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 4 },
        { "date": "2026-07-04T00:00:00Z", "count": 2 }
      ],
      "qaCount": [
        { "date": "2026-07-03T00:00:00Z", "count": 18 },
        { "date": "2026-07-04T00:00:00Z", "count": 21 }
      ]
    },
    "updatedAt": "2026-07-04T08:30:00Z"
  },
  "requestId": "req_admin_metrics_001"
}
```

## Owner 来源

| 聚合字段 | Owner service | 建议来源 |
| --- | --- | --- |
| `userCount` | Auth | `auth_users` 或 Auth 管理员统计接口，排除 `deleted_at IS NOT NULL`。 |
| `knowledgeBaseCount` | Knowledge | Knowledge runtime 或 adapter 聚合的知识库数量。 |
| `documentCount` | Knowledge | Knowledge runtime 或 adapter 聚合的文档数量。 |
| `chunkCount` | Knowledge | Knowledge runtime chunks 或 adapter 暴露的切片统计。 |
| `reportTemplateCount` | Document | `report_templates` 统计，排除软删除记录。 |
| `reportRecordCount` | Document | `reports` 统计，排除软删除记录。 |
| `qaCount` | QA | `messages` 中用户消息或等价 QA 事实聚合，排除已删除会话。 |

Gateway 不直接读取其他服务数据库，不复制 owner service 的内部表结构到公开响应，也不把 prompt、工具参数、原始文档内容、对象存储 key、provider 错误或内部 URL 透出到管理后台。

## 错误语义

| 场景 | 响应 |
| --- | --- |
| 未登录或 access token 无效 | `401 unauthorized` |
| 调用方不是管理员或缺少等价管理权限 | `403 forbidden` |
| `days` 或 `granularity` 不合法 | `400 validation_error` |
| 当前 Gateway 占位实现 | `501 not_implemented` |
| 任一必需 owner service 不可用或返回不可聚合数据 | `502 dependency_error` |

首个后端实现不返回局部成功，也不把缺失来源填成 `0`。如果后续要引入缓存或局部降级，必须先扩展 OpenAPI 和本文档，明确 `stale`、`partial` 和数据新鲜度字段。

## 后续实现验收

- Gateway route 不再返回 `not_implemented`，并保持 `requestId` 贯穿所有 owner 调用。
- Auth、Knowledge、Document 和 QA 的统计口径与本文字段一致。
- `overview` 总览和 `metrics` 趋势不会泄漏内部表结构、对象存储信息、prompt、provider 凭据或原始错误。
- 后端测试覆盖成功聚合、权限拒绝、参数校验、单个 owner 失败映射和 request id 传播。
- 前端只调用 Gateway active path，不直接调用 owner service 统计接口。
