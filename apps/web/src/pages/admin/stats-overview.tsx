import * as echarts from 'echarts/core'
import {
  AlertCircle,
  BarChart3,
  Database,
  FileText,
  MessageSquare,
  Timer,
  Users,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { ApiError } from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EChartsWrapper } from '@/components/ui/echarts'
import { Input } from '@/components/ui/input'
import { useQAMetricsQueries } from '@/features/qa-admin/qa-admin.queries'
import type {
  QAIntentDistributionItem,
  QAMetricsOverview,
  QAMetricsTrendPoint,
  QATopQuery,
} from '@/features/qa-admin/qa-admin.types'

// ── Metric card config ──

type MetricCardConfig = {
  key: keyof QAMetricsOverview
  label: string
  icon: React.ComponentType<{ className?: string; 'aria-hidden'?: boolean | 'true' }>
  format: (value: number) => string
}

const metricCards: MetricCardConfig[] = [
  {
    key: 'totalQaCount',
    label: '总问答次数',
    icon: MessageSquare,
    format: (value) => value.toLocaleString(),
  },
  {
    key: 'todayQaCount',
    label: '今日问答',
    icon: BarChart3,
    format: (value) => value.toLocaleString(),
  },
  {
    key: 'totalQuestionCount',
    label: '问题总数',
    icon: MessageSquare,
    format: (value) => value.toLocaleString(),
  },
  {
    key: 'conversationCount',
    label: '会话数',
    icon: Users,
    format: (value) => value.toLocaleString(),
  },
  {
    key: 'avgLatencyMs',
    label: '平均延迟',
    icon: Timer,
    format: (value) => `${Math.round(value)} ms`,
  },
  {
    key: 'activeUsersToday',
    label: '今日活跃用户',
    icon: Users,
    format: (value) => value.toLocaleString(),
  },
  {
    key: 'knowledgeBaseCount',
    label: '知识库数量',
    icon: Database,
    format: (value) => value.toLocaleString(),
  },
  {
    key: 'documentCount',
    label: '文档总数',
    icon: FileText,
    format: (value) => value.toLocaleString(),
  },
]

// ── Helpers ──

function isDark(): boolean {
  if (typeof document === 'undefined') return false
  return document.documentElement.classList.contains('dark')
}

function chartTextColor(): string {
  return isDark() ? '#a1a1aa' : '#71717a'
}

function chartBorderColor(): string {
  return isDark() ? '#27272a' : '#e4e4e7'
}

function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.requestId ? `${error.message}（requestId: ${error.requestId}）` : error.message
  }
  return error instanceof Error ? error.message : '未知错误'
}

function formatDate(value: string | undefined): string {
  return value ? new Date(value).toLocaleDateString() : '-'
}

function formatNumber(value: number | undefined): string {
  return value === undefined ? '-' : value.toLocaleString()
}

// ── Shared UI ──

function SectionState({ message, tone }: { message: string; tone: 'empty' | 'error' }) {
  return (
    <div
      className={
        tone === 'error'
          ? 'flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive'
          : 'flex items-start gap-2 rounded-lg border border-dashed border-border bg-muted/20 p-4 text-sm text-muted-foreground'
      }
    >
      <AlertCircle aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <span>{message}</span>
    </div>
  )
}

function MetricCardSkeleton() {
  return (
    <div className="h-28 rounded-lg border border-border bg-card p-4">
      <div className="mb-4 h-4 w-24 rounded skeleton-shimmer" />
      <div className="h-7 w-20 rounded skeleton-shimmer" />
    </div>
  )
}

function MetricCard({
  config,
  overview,
}: {
  config: MetricCardConfig
  overview: QAMetricsOverview
}) {
  const Icon = config.icon
  const rawValue = overview[config.key]
  const unavailable = rawValue === undefined || rawValue === null

  return (
    <div className="rounded-lg border border-border bg-card p-4 hover:-translate-y-0.5 hover:shadow-md transition-all duration-200">
      <div className="mb-2 flex items-center justify-between gap-2 text-sm text-muted-foreground">
        <span className="flex items-center gap-2">
          <Icon aria-hidden="true" className="size-4" />
          {config.label}
        </span>
        {unavailable && <Badge variant="outline">不可用</Badge>}
      </div>
      <p className="text-2xl font-semibold text-foreground">
        {unavailable ? '-' : config.format(rawValue)}
      </p>
    </div>
  )
}

// ── ECharts-based charts ──

function TrendChart({ points }: { points: QAMetricsTrendPoint[] }) {
  const normalizedPoints = points.map((point) => ({
    date: point.date,
    count: point.count ?? point.questionCount ?? 0,
  }))

  const textColor = chartTextColor()
  const borderColor = chartBorderColor()

  const option = useMemo(
    () => ({
      grid: { top: 16, right: 16, bottom: 24, left: 48 },
      xAxis: {
        type: 'category' as const,
        data: normalizedPoints.map((p) => p.date),
        axisLine: { lineStyle: { color: borderColor } },
        axisTick: { show: false },
        axisLabel: { color: textColor, fontSize: 11 },
      },
      yAxis: {
        type: 'value' as const,
        splitLine: { lineStyle: { color: borderColor, type: 'dashed' as const } },
        axisLabel: { color: textColor, fontSize: 11 },
      },
      tooltip: {
        trigger: 'axis' as const,
        backgroundColor: isDark() ? '#18181b' : '#fff',
        borderColor,
        textStyle: { fontSize: 13 },
      },
      series: [
        {
          name: '问答数量',
          type: 'line',
          data: normalizedPoints.map((p) => p.count),
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2, color: '#6366f1' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(99,102,241,0.25)' },
              { offset: 1, color: 'rgba(99,102,241,0.02)' },
            ]),
          },
        },
      ],
    }),
    [normalizedPoints, textColor, borderColor],
  )

  return <EChartsWrapper option={option} style={{ minHeight: 280 }} />
}

function IntentRoseChart({ items }: { items: QAIntentDistributionItem[] }) {
  const textColor = chartTextColor()
  const data = items.map((item) => ({
    name: item.label ?? item.intent,
    value: item.count,
  }))

  const option = useMemo(
    () => ({
      tooltip: {
        trigger: 'item' as const,
        backgroundColor: isDark() ? '#18181b' : '#fff',
        borderColor: chartBorderColor(),
        textStyle: { fontSize: 13 },
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'vertical' as const,
        left: 0,
        top: 'center',
        textStyle: { color: textColor, fontSize: 12 },
        itemWidth: 10,
        itemHeight: 10,
        itemGap: 12,
      },
      series: [
        {
          name: '意图分布',
          type: 'pie',
          radius: ['20%', '75%'],
          center: ['58%', '50%'],
          roseType: 'area',
          itemStyle: { borderRadius: 4, borderColor: isDark() ? '#18181b' : '#fff', borderWidth: 2 },
          label: {
            color: textColor,
            fontSize: 11,
            formatter: '{b}\n{d}%',
          },
          emphasis: {
            label: { fontSize: 14, fontWeight: 'bold' },
            itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.15)' },
          },
          data,
        },
      ],
    }),
    [data, textColor],
  )

  return <EChartsWrapper option={option} style={{ minHeight: 300 }} />
}

// ── Table ──

function TopQueriesTable({ queries }: { queries: QATopQuery[] }) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-left text-sm">
        <thead className="bg-muted/40 text-xs text-muted-foreground">
          <tr>
            <th className="px-3 py-2 font-medium">问题</th>
            <th className="w-24 px-3 py-2 font-medium">次数</th>
            <th className="w-28 px-3 py-2 font-medium">平均延迟</th>
            <th className="w-36 px-3 py-2 font-medium">最近提问</th>
          </tr>
        </thead>
        <tbody>
          {queries.map((query) => (
            <tr
              key={`${query.query}-${query.lastAskedAt ?? ''}`}
              className="border-t border-border transition-colors duration-150 hover:bg-muted/30"
            >
              <td className="break-words px-3 py-2 text-foreground">{query.query}</td>
              <td className="px-3 py-2 font-mono">{query.count}</td>
              <td className="px-3 py-2 font-mono">
                {query.avgLatencyMs === undefined ? '-' : `${query.avgLatencyMs}ms`}
              </td>
              <td className="px-3 py-2 text-xs text-muted-foreground">
                {formatDate(query.lastAskedAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ── Page ──

export function StatsOverviewPage() {
  const [overviewDays, setOverviewDays] = useState('1')
  const [trendDays, setTrendDays] = useState('30')
  const [rankingDays, setRankingDays] = useState('7')
  const [rankingLimit, setRankingLimit] = useState('10')
  const [, setDarkModeKey] = useState(0)

  // Re-render charts on theme change
  useEffect(() => {
    const observer = new MutationObserver(() => setDarkModeKey((k) => k + 1))
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  const filters = useMemo(
    () => ({
      overviewDays: Math.max(1, Number(overviewDays) || 1),
      trendDays: Math.max(1, Number(trendDays) || 30),
      rankingDays: Math.max(1, Number(rankingDays) || 7),
      rankingLimit: Math.max(1, Number(rankingLimit) || 10),
    }),
    [overviewDays, rankingDays, rankingLimit, trendDays],
  )

  const { overviewQuery, trendQuery, topQueriesQuery, intentDistributionQuery } =
    useQAMetricsQueries(filters)

  const refreshAll = () => {
    void overviewQuery.refetch()
    void trendQuery.refetch()
    void topQueriesQuery.refetch()
    void intentDistributionQuery.refetch()
  }

  const trendPoints = trendQuery.data?.points ?? trendQuery.data?.trend30d ?? []
  const isFetching =
    overviewQuery.isFetching ||
    trendQuery.isFetching ||
    topQueriesQuery.isFetching ||
    intentDistributionQuery.isFetching

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-2xl font-semibold text-foreground">QA 统计</h3>
          <p className="mt-2 text-sm text-muted-foreground">
            展示概览指标、趋势、热门问题和意图分布；缺失的知识指标会标记为不可用。
          </p>
        </div>
        <Button type="button" variant="outline" onClick={refreshAll} disabled={isFetching}>
          刷新
        </Button>
      </div>

      {/* Metric cards */}
      <section className="space-y-4">
        <div className="flex flex-wrap items-end gap-3">
          <label className="w-32 space-y-1 text-sm">
            <span className="font-medium text-foreground">概览天数</span>
            <Input
              value={overviewDays}
              inputMode="numeric"
              onChange={(event) => setOverviewDays(event.target.value)}
            />
          </label>
        </div>

        {overviewQuery.isError ? (
          <SectionState
            tone="error"
            message={`概览指标加载失败：${getErrorMessage(overviewQuery.error)}`}
          />
        ) : overviewQuery.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 8 }).map((_, index) => (
              <MetricCardSkeleton key={index} />
            ))}
          </div>
        ) : overviewQuery.data ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {metricCards.map((config) => (
              <MetricCard key={config.key} config={config} overview={overviewQuery.data} />
            ))}
          </div>
        ) : (
          <SectionState tone="empty" message="暂无概览指标。" />
        )}
      </section>

      {/* Trend chart + Intent rose chart */}
      <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <div className="space-y-4 rounded-lg border border-border bg-card p-5">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h4 className="text-lg font-semibold text-foreground">趋势</h4>
              <p className="mt-1 text-sm text-muted-foreground">按日期展示问答数量。</p>
            </div>
            <label className="w-28 space-y-1 text-sm">
              <span className="font-medium text-foreground">天数</span>
              <Input
                value={trendDays}
                inputMode="numeric"
                onChange={(event) => setTrendDays(event.target.value)}
              />
            </label>
          </div>
          {trendQuery.isError ? (
            <SectionState
              tone="error"
              message={`趋势加载失败：${getErrorMessage(trendQuery.error)}`}
            />
          ) : trendQuery.isLoading ? (
            <div className="h-72 skeleton-shimmer rounded-lg" />
          ) : trendPoints.length === 0 ? (
            <SectionState tone="empty" message="当前窗口内暂无趋势数据。" />
          ) : (
            <TrendChart points={trendPoints} />
          )}
        </div>

        <div className="space-y-4 rounded-lg border border-border bg-card p-5">
          <div>
            <h4 className="text-lg font-semibold text-foreground">意图分布</h4>
            <p className="mt-1 text-sm text-muted-foreground">南丁格尔玫瑰图 · 按问答意图聚合占比。</p>
          </div>
          {intentDistributionQuery.isError ? (
            <SectionState
              tone="error"
              message={`意图分布加载失败：${getErrorMessage(intentDistributionQuery.error)}`}
            />
          ) : intentDistributionQuery.isLoading ? (
            <div className="h-72 skeleton-shimmer rounded-lg" />
          ) : (intentDistributionQuery.data ?? []).length === 0 ? (
            <SectionState tone="empty" message="当前窗口内暂无意图分布数据。" />
          ) : (
            <IntentRoseChart items={intentDistributionQuery.data ?? []} />
          )}
        </div>
      </section>

      {/* Top queries */}
      <section className="space-y-4 rounded-lg border border-border bg-card p-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h4 className="text-lg font-semibold text-foreground">热门问题</h4>
            <p className="mt-1 text-sm text-muted-foreground">按提问次数排序展示。</p>
          </div>
          <div className="flex gap-3">
            <label className="w-28 space-y-1 text-sm">
              <span className="font-medium text-foreground">天数</span>
              <Input
                value={rankingDays}
                inputMode="numeric"
                onChange={(event) => setRankingDays(event.target.value)}
              />
            </label>
            <label className="w-28 space-y-1 text-sm">
              <span className="font-medium text-foreground">条数</span>
              <Input
                value={rankingLimit}
                inputMode="numeric"
                onChange={(event) => setRankingLimit(event.target.value)}
              />
            </label>
          </div>
        </div>

        {topQueriesQuery.isError ? (
          <SectionState
            tone="error"
            message={`热门问题加载失败：${getErrorMessage(topQueriesQuery.error)}`}
          />
        ) : topQueriesQuery.isLoading ? (
          <div className="h-60 skeleton-shimmer rounded-lg" />
        ) : (topQueriesQuery.data ?? []).length === 0 ? (
          <SectionState tone="empty" message="当前窗口内暂无热门问题。" />
        ) : (
          <TopQueriesTable queries={topQueriesQuery.data ?? []} />
        )}

        <div className="text-xs text-muted-foreground">
          当前参数：overview {formatNumber(filters.overviewDays)} 天，trend{' '}
          {formatNumber(filters.trendDays)} 天，ranking {formatNumber(filters.rankingDays)} 天。
        </div>
      </section>
    </div>
  )
}
