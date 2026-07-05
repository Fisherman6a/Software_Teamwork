import { Download, Eye, FileText, Loader2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  useCreateReportFileMutation,
  useReportFileQuery,
  useReportJobQuery,
} from '@/features/reports'
import type { QAReportArtifact } from '@/lib/types'
import { cn } from '@/lib/utils'

import { ReportArtifactEditorDialog } from './report-artifact-editor-dialog'

type ReportArtifactCardProps = {
  artifact: QAReportArtifact
  onDownload?: (downloadPath: string, filename: string) => void
}

function getJobStatusVariant(
  status: string | undefined,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'succeeded' || status === 'completed') return 'default'
  if (status === 'failed') return 'destructive'
  return 'secondary'
}

function getJobStatusLabel(status: string | undefined): string {
  switch (status) {
    case 'accepted':
      return '已接受'
    case 'pending':
      return '等待中'
    case 'running':
      return '生成中'
    case 'succeeded':
      return '已完成'
    case 'completed':
      return '已完成'
    case 'partial_succeeded':
      return '部分成功'
    case 'failed':
      return '失败'
    case 'canceled':
      return '已取消'
    default:
      return status ?? '处理中'
  }
}

function isJobRunning(status: string | undefined): boolean {
  return status === 'running' || status === 'accepted' || status === 'pending'
}

function isJobFailed(status: string | undefined): boolean {
  return status === 'failed'
}

function canDownload(artifact: QAReportArtifact): boolean {
  return artifact.fileStatus === 'succeeded' && Boolean(artifact.downloadPath)
}

function reportFileDownloadPath(reportFileId: string): string {
  return `/api/v1/report-files/${reportFileId}/content`
}

function isSuccessfulJobStatus(status: string | undefined): boolean {
  return status === 'succeeded' || status === 'partial_succeeded' || status === 'completed'
}

function readProgressInteger(
  progress: Record<string, unknown> | undefined,
  keys: string[],
): number | null {
  if (!progress) return null
  for (const key of keys) {
    const value = progress[key]
    if (typeof value !== 'number' || !Number.isFinite(value)) continue
    return Math.max(0, Math.trunc(value))
  }
  return null
}

function getProgressCounts(progress: Record<string, unknown> | undefined): {
  completed: number
  total: number
} | null {
  const completed = readProgressInteger(progress, ['completed', 'completedSections'])
  const total = readProgressInteger(progress, ['total', 'totalSections'])
  if (completed == null || total == null || total <= 0) return null
  return {
    completed: Math.min(completed, total),
    total,
  }
}

const MAX_TITLES = 5
const MAX_SUMMARY_LENGTH = 120

export default function ReportArtifactCard({ artifact, onDownload }: ReportArtifactCardProps) {
  const [createdReportFileId, setCreatedReportFileId] = useState<string | null>(null)
  const [editorOpen, setEditorOpen] = useState(false)
  const downloadedReportFileIdsRef = useRef(new Set<string>())
  const createReportFileMutation = useCreateReportFileMutation()
  const shouldPollJob =
    Boolean(artifact.jobId) &&
    (artifact.jobStatus === 'accepted' ||
      artifact.jobStatus === 'pending' ||
      artifact.jobStatus === 'running')
  const jobQuery = useReportJobQuery(shouldPollJob ? (artifact.jobId ?? null) : null)
  const latestJob = jobQuery.data
  const jobStatus = latestJob?.status ?? artifact.jobStatus
  const progressCounts = getProgressCounts(latestJob?.progress)
  const reportFileId = artifact.reportFileId ?? createdReportFileId
  const reportFileQuery = useReportFileQuery(reportFileId)
  const createdReportFile = reportFileQuery.data
  const createdDownloadPath =
    createdReportFile?.contentPath ??
    (createdReportFile?.status === 'succeeded'
      ? reportFileDownloadPath(createdReportFile.id)
      : undefined)
  const reportName = artifact.reportName ?? '报告生成'
  const preview = artifact.preview

  // Collect display titles from preview
  const titles = preview?.outlineTitles ?? preview?.sectionTitles ?? []
  const displayTitles = titles.slice(0, MAX_TITLES)
  const remaining = titles.length - MAX_TITLES

  const summary = preview?.summary ?? ''
  const truncatedSummary =
    summary.length > MAX_SUMMARY_LENGTH ? summary.slice(0, MAX_SUMMARY_LENGTH) + '…' : summary
  const isExporting =
    createReportFileMutation.isPending ||
    createdReportFile?.status === 'pending' ||
    createdReportFile?.status === 'running'
  const exportFailed = createReportFileMutation.isError || createdReportFile?.status === 'failed'
  const canCreateDownload =
    !canDownload(artifact) &&
    !createdDownloadPath &&
    Boolean(artifact.reportId) &&
    isSuccessfulJobStatus(jobStatus) &&
    !isExporting
  const canClickDownload =
    canDownload(artifact) || Boolean(createdDownloadPath) || canCreateDownload
  const canOpenEditor = Boolean(artifact.reportId)

  useEffect(() => {
    if (!createdReportFile || createdReportFile.status !== 'succeeded') return
    if (!createdDownloadPath) return
    if (downloadedReportFileIdsRef.current.has(createdReportFile.id)) return
    downloadedReportFileIdsRef.current.add(createdReportFile.id)
    onDownload?.(
      createdDownloadPath,
      createdReportFile.filename ?? artifact.filename ?? 'report.docx',
    )
  }, [artifact.filename, createdDownloadPath, createdReportFile, onDownload])

  const handleDownload = async () => {
    if (canDownload(artifact) && artifact.downloadPath) {
      onDownload?.(artifact.downloadPath, artifact.filename ?? 'report.docx')
      return
    }
    if (createdDownloadPath) {
      onDownload?.(
        createdDownloadPath,
        createdReportFile?.filename ?? artifact.filename ?? 'report.docx',
      )
      return
    }
    if (!canCreateDownload || !artifact.reportId) return
    let file
    try {
      file = await createReportFileMutation.mutateAsync({
        reportId: artifact.reportId,
        format: 'docx',
        styleOptions: { numberingMode: 'global' },
      })
    } catch {
      return
    }
    setCreatedReportFileId(file.id)
    if (file.status === 'succeeded' && file.contentPath) {
      downloadedReportFileIdsRef.current.add(file.id)
      onDownload?.(file.contentPath, file.filename ?? artifact.filename ?? 'report.docx')
    }
  }

  const actionLabel = canDownload(artifact)
    ? '下载报告'
    : createdDownloadPath
      ? '下载报告'
      : isExporting
        ? '导出中'
        : exportFailed
          ? '重试导出'
          : canCreateDownload
            ? '生成并下载'
            : '下载报告'

  return (
    <div className="mt-3 overflow-hidden rounded-lg border border-border bg-card shadow-sm transition-shadow hover:shadow-md">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border/50 px-4 py-3">
        <div className="flex items-center gap-2 min-w-0">
          <FileText className="size-4 shrink-0 text-muted-foreground" />
          <span className="truncate text-sm font-medium">{reportName}</span>
          {preview?.statusText && (
            <span className="truncate text-xs text-muted-foreground">{preview.statusText}</span>
          )}
        </div>
        {jobStatus && (
          <Badge
            variant={getJobStatusVariant(jobStatus)}
            className={cn('ml-2 shrink-0', isJobRunning(jobStatus) && 'animate-pulse')}
          >
            {getJobStatusLabel(jobStatus)}
          </Badge>
        )}
      </div>

      {/* Preview body */}
      {(displayTitles.length > 0 ||
        truncatedSummary ||
        preview?.progressPercent != null ||
        progressCounts ||
        exportFailed) && (
        <div className="px-4 py-3 space-y-2">
          {progressCounts && (
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>生成进度</span>
              <span>
                {progressCounts.completed}/{progressCounts.total}
              </span>
            </div>
          )}

          {/* Progress bar */}
          {preview?.progressPercent != null && (
            <div className="w-full">
              <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
                <span>进度</span>
                <span>{preview.progressPercent}%</span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className={cn(
                    'h-full rounded-full transition-all duration-500',
                    isJobFailed(jobStatus) ? 'bg-destructive' : 'bg-primary',
                  )}
                  style={{ width: `${Math.min(100, Math.max(0, preview.progressPercent))}%` }}
                />
              </div>
            </div>
          )}

          {/* Titles */}
          {displayTitles.length > 0 && (
            <div className="space-y-0.5">
              {displayTitles.map((t, i) => (
                <div key={i} className="flex items-start gap-1.5 text-xs text-muted-foreground">
                  <span className="mt-0.5 size-1 shrink-0 rounded-full bg-muted-foreground/40" />
                  <span className="truncate">{t}</span>
                </div>
              ))}
              {remaining > 0 && (
                <div className="text-xs text-muted-foreground pl-3.5">...等 {remaining} 项</div>
              )}
            </div>
          )}

          {/* Summary */}
          {truncatedSummary && (
            <p className="text-xs leading-relaxed text-muted-foreground">{truncatedSummary}</p>
          )}

          {exportFailed && (
            <p className="text-xs leading-relaxed text-destructive">报告文件导出失败，请重试。</p>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="flex flex-wrap items-center justify-end gap-2 border-t border-border/50 px-4 py-2">
        <Button
          variant="outline"
          size="sm"
          disabled={!canOpenEditor}
          onClick={() => setEditorOpen(true)}
          className="h-7 px-2 text-xs transition-all hover:bg-primary hover:text-primary-foreground hover:scale-105 active:scale-95"
        >
          <Eye className="mr-1 size-3" />
          查看编辑
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={!canClickDownload}
          onClick={() => void handleDownload()}
          className="h-7 px-2 text-xs transition-all hover:bg-primary hover:text-primary-foreground hover:scale-105 active:scale-95"
        >
          {isExporting ? (
            <Loader2 className="mr-1 size-3 animate-spin" />
          ) : (
            <Download className="mr-1 size-3" />
          )}
          {actionLabel}
        </Button>
      </div>
      <ReportArtifactEditorDialog
        artifact={artifact}
        open={editorOpen}
        onOpenChange={setEditorOpen}
      />
    </div>
  )
}
