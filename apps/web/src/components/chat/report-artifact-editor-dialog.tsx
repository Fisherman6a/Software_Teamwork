import { Eye, Loader2, Save } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'

import { InlineNotice, StateBlock } from '@/components/common'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  formatReportGatewayError,
  type ReportSection,
  ReportSectionTables,
  useReport,
  useReportDetailQueries,
  useUpdateReportSectionMutation,
} from '@/features/reports'
import type { QAReportArtifact } from '@/lib/types'
import { cn } from '@/lib/utils'

type ReportArtifactEditorDialogProps = {
  artifact: QAReportArtifact
  onOpenChange: (open: boolean) => void
  open: boolean
}

type SectionDraftState = {
  content: string
  sectionId: string
  tables: Record<string, unknown>[]
  title: string
}

function sectionLabel(section: ReportSection): string {
  return section.numbering ? `${section.numbering} ${section.title}` : section.title
}

function isDraftDirty(section: ReportSection | undefined, draft: SectionDraftState | null) {
  if (!section || !draft || draft.sectionId !== section.id) return false
  return (
    draft.title !== section.title ||
    draft.content !== (section.content ?? '') ||
    JSON.stringify(draft.tables) !== JSON.stringify(section.tables ?? [])
  )
}

export function ReportArtifactEditorDialog({
  artifact,
  onOpenChange,
  open,
}: ReportArtifactEditorDialogProps) {
  const reportId = artifact.reportId ?? null
  const [activeSectionId, setActiveSectionId] = useState('')
  const [draft, setDraft] = useState<SectionDraftState | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const wasOpenRef = useRef(open)

  const reportQuery = useReport(open ? reportId : null)
  const { sectionsQuery } = useReportDetailQueries(open ? reportId : null)
  const saveSectionMutation = useUpdateReportSectionMutation(reportId ?? '')
  const resetSaveSectionMutation = saveSectionMutation.reset

  const sections = useMemo(() => sectionsQuery.data ?? [], [sectionsQuery.data])
  const activeSection =
    sections.find((section) => section.id === activeSectionId) ?? sections[0] ?? undefined
  const dirty = isDraftDirty(activeSection, draft)
  const canEdit = Boolean(reportId)

  useEffect(() => {
    const wasOpen = wasOpenRef.current
    wasOpenRef.current = open

    if (open || !wasOpen) return

    setActiveSectionId('')
    setDraft(null)
    setNotice(null)
    resetSaveSectionMutation()
  }, [open, resetSaveSectionMutation])

  useEffect(() => {
    if (!open || sections.length === 0) return
    if (activeSectionId && sections.some((section) => section.id === activeSectionId)) return
    setActiveSectionId(sections[0]?.id ?? '')
  }, [activeSectionId, open, sections])

  useEffect(() => {
    if (!activeSection) {
      setDraft(null)
      return
    }
    setDraft({
      content: activeSection.content ?? '',
      sectionId: activeSection.id,
      tables: activeSection.tables ?? [],
      title: activeSection.title,
    })
  }, [activeSection])

  const handleSelectSection = (sectionId: string) => {
    if (sectionId === activeSectionId) return
    if (dirty) {
      setNotice('当前章节有未保存修改，请先保存后再切换章节。')
      return
    }
    setNotice(null)
    setActiveSectionId(sectionId)
  }

  const handleSave = async () => {
    if (!reportId || !activeSection || !draft) return

    setNotice(null)
    try {
      await saveSectionMutation.mutateAsync({
        content: draft.content,
        sectionId: activeSection.id,
        tables: draft.tables,
        title: draft.title.trim() || activeSection.title,
      })
      setNotice('章节已保存。')
    } catch (error) {
      setNotice(formatReportGatewayError(error, '章节保存失败'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!w-[min(1180px,calc(100vw-2rem))] !max-w-none h-[calc(100vh-2rem)] max-h-[calc(100vh-2rem)] grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden p-0 sm:!max-w-none">
        <DialogHeader className="border-b border-border px-5 py-4">
          <DialogTitle className="flex min-w-0 items-center gap-2">
            <Eye className="size-4 shrink-0 text-muted-foreground" />
            <span className="truncate">
              {artifact.reportName ?? reportQuery.data?.name ?? '报告文档'}
            </span>
          </DialogTitle>
          <DialogDescription>
            查看并编辑报告生成工具产出的服务端报告章节，保存后会更新报告正文数据。
          </DialogDescription>
        </DialogHeader>

        {!canEdit ? (
          <div className="p-5">
            <StateBlock
              description="这个 QA artifact 没有返回 reportId，前端只能下载已有文件，不能打开服务端章节编辑。"
              size="compact"
              title="缺少报告草稿"
              variant="warning"
            />
          </div>
        ) : reportQuery.isLoading || sectionsQuery.isLoading ? (
          <div className="p-5">
            <StateBlock size="compact" title="报告加载中" variant="loading" />
          </div>
        ) : reportQuery.isError || sectionsQuery.isError ? (
          <div className="p-5">
            <StateBlock
              description={
                reportQuery.isError
                  ? formatReportGatewayError(reportQuery.error, '报告加载失败')
                  : formatReportGatewayError(sectionsQuery.error, '章节加载失败')
              }
              size="compact"
              title="无法打开报告"
              variant="error"
            />
          </div>
        ) : sections.length === 0 ? (
          <div className="p-5">
            <StateBlock
              description="报告生成任务完成正文生成后，章节会在这里显示。"
              size="compact"
              title="暂无章节"
              variant="empty"
            />
          </div>
        ) : (
          <div className="grid min-h-0 min-w-0 grid-cols-1 overflow-hidden lg:grid-cols-[minmax(260px,340px)_minmax(0,1fr)]">
            <aside className="min-h-0 min-w-0 border-b border-border bg-muted/30 p-4 lg:border-r lg:border-b-0">
              <div className="mb-3 flex items-center justify-between gap-2">
                <h3 className="text-sm font-medium">文档章节</h3>
                <Badge variant="outline">{sections.length}</Badge>
              </div>
              <div className="max-h-[220px] space-y-2 overflow-y-auto pr-1 lg:max-h-[calc(100vh-18rem)]">
                {sections.map((section) => (
                  <button
                    key={section.id}
                    className={cn(
                      'flex w-full items-start justify-between gap-2 rounded-lg border px-3 py-2 text-left text-sm transition-colors',
                      activeSection?.id === section.id
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border bg-background text-muted-foreground hover:text-foreground',
                    )}
                    type="button"
                    onClick={() => handleSelectSection(section.id)}
                  >
                    <span className="min-w-0 flex-1 truncate">{sectionLabel(section)}</span>
                    {section.manualEdited && (
                      <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[0.7rem] text-muted-foreground">
                        已编辑
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </aside>

            <main className="flex min-h-0 min-w-0 flex-col gap-4 overflow-hidden p-4">
              <div className="min-h-[50px] shrink-0">
                {notice && (
                  <InlineNotice
                    className="py-2.5"
                    title={saveSectionMutation.isError ? '保存失败' : undefined}
                    variant={saveSectionMutation.isError ? 'error' : 'info'}
                  >
                    {notice}
                  </InlineNotice>
                )}
              </div>

              {activeSection && draft && (
                <>
                  <label className="shrink-0 space-y-2">
                    <span className="text-sm font-medium">章节标题</span>
                    <Input
                      value={draft.title}
                      onChange={(event) =>
                        setDraft((prev) => (prev ? { ...prev, title: event.target.value } : prev))
                      }
                    />
                  </label>

                  <label className="flex min-h-[220px] shrink-0 flex-col gap-2">
                    <span className="text-sm font-medium">章节正文</span>
                    <Textarea
                      className="h-[clamp(220px,32vh,360px)] !field-sizing-fixed resize-none overflow-y-auto text-sm leading-7"
                      maxLength={50000}
                      value={draft.content}
                      onChange={(event) =>
                        setDraft((prev) => (prev ? { ...prev, content: event.target.value } : prev))
                      }
                    />
                  </label>

                  <ReportSectionTables
                    className="min-h-0 flex-1 border-t border-border pt-4"
                    editable
                    tables={draft.tables}
                    onChange={(tables) => setDraft((prev) => (prev ? { ...prev, tables } : prev))}
                  />
                </>
              )}
            </main>
          </div>
        )}

        <DialogFooter className="px-5">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
          <Button
            disabled={!activeSection || !draft || !dirty || saveSectionMutation.isPending}
            onClick={() => void handleSave()}
          >
            {saveSectionMutation.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Save className="size-4" />
            )}
            保存章节
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
