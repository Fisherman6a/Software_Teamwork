import { Minus, Plus, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

export type RenderableReportTable = {
  caption?: string
  footnote?: string
  headers: string[]
  rows: string[][]
}

function tableCellText(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function readStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.map(tableCellText)
}

function readTableRows(value: unknown): string[][] {
  if (!Array.isArray(value)) return []
  return value.map((row) => (Array.isArray(row) ? row.map(tableCellText) : [tableCellText(row)]))
}

function getFirstStringField(table: Record<string, unknown>, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = table[key]
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return undefined
}

function normalizeReportTable(table: Record<string, unknown>): RenderableReportTable | null {
  const headers = readStringArray(table.headers ?? table.columns)
  const rows = readTableRows(table.rows ?? table.data)
  if (headers.length === 0 && rows.length === 0) return null

  return {
    caption: getFirstStringField(table, ['caption', 'title', 'name']),
    footnote: getFirstStringField(table, ['footnote', 'note', 'remark']),
    headers,
    rows,
  }
}

export function normalizeReportTables(
  tables: Record<string, unknown>[] | undefined,
): RenderableReportTable[] {
  return (tables ?? []).flatMap((table) => {
    const normalized = normalizeReportTable(table)
    return normalized ? [normalized] : []
  })
}

function reportTablesToRecords(tables: RenderableReportTable[]): Record<string, unknown>[] {
  return tables.map((table) => ({
    ...(table.caption ? { caption: table.caption } : {}),
    ...(table.footnote ? { footnote: table.footnote } : {}),
    headers: table.headers,
    rows: table.rows,
  }))
}

function createEmptyTable(): RenderableReportTable {
  return {
    headers: ['列 1', '列 2'],
    rows: [['', '']],
  }
}

function getColumnCount(table: RenderableReportTable): number {
  return Math.max(table.headers.length, ...table.rows.map((row) => row.length), 1)
}

type ReportSectionTablesProps = {
  className?: string
  editable?: boolean
  onChange?: (tables: Record<string, unknown>[]) => void
  tables: Record<string, unknown>[] | undefined
  title?: string
}

export function ReportSectionTables({
  className,
  editable = false,
  onChange,
  tables,
  title = '章节表格',
}: ReportSectionTablesProps) {
  const normalizedTables = normalizeReportTables(tables)
  const updateTables = (updater: (tables: RenderableReportTable[]) => RenderableReportTable[]) => {
    onChange?.(reportTablesToRecords(updater(normalizedTables)))
  }

  const updateTable = (
    tableIndex: number,
    updater: (table: RenderableReportTable) => RenderableReportTable,
  ) => {
    updateTables((currentTables) =>
      currentTables.map((table, index) => (index === tableIndex ? updater(table) : table)),
    )
  }

  const addTable = () => {
    updateTables((currentTables) => [...currentTables, createEmptyTable()])
  }

  if (normalizedTables.length === 0) {
    if (!editable) return null

    return (
      <div aria-label={title} className={cn('flex min-h-0 flex-col', className)}>
        <div className="flex shrink-0 items-center justify-between gap-3">
          <h4 className="text-sm font-medium">{title}</h4>
          <Button type="button" size="sm" variant="outline" onClick={addTable}>
            <Plus className="size-3.5" />
            新增表格
          </Button>
        </div>
        <p className="mt-3 rounded-lg border border-dashed border-border bg-background px-3 py-4 text-sm text-muted-foreground">
          暂无表格。
        </p>
      </div>
    )
  }

  return (
    <div aria-label={title} className={cn('flex min-h-0 flex-col', className)}>
      <div className="flex shrink-0 items-center justify-between gap-3">
        <h4 className="text-sm font-medium">{title}</h4>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{normalizedTables.length} 张</span>
          {editable && (
            <Button type="button" size="sm" variant="outline" onClick={addTable}>
              <Plus className="size-3.5" />
              新增表格
            </Button>
          )}
        </div>
      </div>
      <div className="mt-3 min-h-0 space-y-3 overflow-y-auto pr-1">
        {normalizedTables.map((table, tableIndex) => {
          const columnCount = getColumnCount(table)
          const setCaption = (caption: string) => {
            updateTable(tableIndex, (current) => ({ ...current, caption }))
          }
          const setFootnote = (footnote: string) => {
            updateTable(tableIndex, (current) => ({ ...current, footnote }))
          }
          const setHeader = (headerIndex: number, value: string) => {
            updateTable(tableIndex, (current) => {
              const headers = Array.from(
                { length: getColumnCount(current) },
                (_, index) => current.headers[index] ?? `列 ${index + 1}`,
              )
              headers[headerIndex] = value
              return { ...current, headers }
            })
          }
          const setCell = (rowIndex: number, cellIndex: number, value: string) => {
            updateTable(tableIndex, (current) => {
              const count = getColumnCount(current)
              const rows = current.rows.map((row, index) => {
                const nextRow = Array.from(
                  { length: count },
                  (_, columnIndex) => row[columnIndex] ?? '',
                )
                if (index === rowIndex) nextRow[cellIndex] = value
                return nextRow
              })
              return { ...current, rows }
            })
          }
          const addRow = () => {
            updateTable(tableIndex, (current) => ({
              ...current,
              rows: [...current.rows, Array.from({ length: getColumnCount(current) }, () => '')],
            }))
          }
          const removeRow = (rowIndex: number) => {
            updateTable(tableIndex, (current) => ({
              ...current,
              rows: current.rows.filter((_, index) => index !== rowIndex),
            }))
          }
          const addColumn = () => {
            updateTable(tableIndex, (current) => {
              const count = getColumnCount(current)
              return {
                ...current,
                headers: [...current.headers, `列 ${count + 1}`],
                rows: current.rows.map((row) => [...row, '']),
              }
            })
          }
          const removeColumn = (columnIndex: number) => {
            if (columnCount <= 1) return
            updateTable(tableIndex, (current) => ({
              ...current,
              headers: current.headers.filter((_, index) => index !== columnIndex),
              rows: current.rows.map((row) => row.filter((_, index) => index !== columnIndex)),
            }))
          }
          const removeTable = () => {
            updateTables((currentTables) =>
              currentTables.filter((_, index) => index !== tableIndex),
            )
          }

          return (
            <div
              className="overflow-hidden rounded-lg border border-border bg-background"
              key={`${table.caption ?? 'table'}-${tableIndex}`}
            >
              {(editable || table.caption) && (
                <div className="flex items-center gap-2 border-b border-border bg-muted/40 px-3 py-2">
                  {editable ? (
                    <Input
                      aria-label={`表格 ${tableIndex + 1} 标题`}
                      className="h-8 min-w-0 flex-1 bg-background text-sm font-medium"
                      placeholder="表格标题"
                      value={table.caption ?? ''}
                      onChange={(event) => setCaption(event.target.value)}
                    />
                  ) : (
                    <span className="min-w-0 flex-1 text-sm font-medium">{table.caption}</span>
                  )}
                  {editable && (
                    <Button
                      aria-label={`删除表格 ${tableIndex + 1}`}
                      type="button"
                      size="icon"
                      variant="ghost"
                      className="size-8 shrink-0"
                      onClick={removeTable}
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  )}
                </div>
              )}
              {editable && (
                <div className="flex flex-wrap gap-2 border-b border-border px-3 py-2">
                  <Button type="button" size="sm" variant="outline" onClick={addRow}>
                    <Plus className="size-3.5" />
                    新增行
                  </Button>
                  <Button type="button" size="sm" variant="outline" onClick={addColumn}>
                    <Plus className="size-3.5" />
                    新增列
                  </Button>
                </div>
              )}
              <div className="overflow-x-auto">
                <table className="w-full min-w-[420px] border-collapse text-sm">
                  <thead>
                    <tr className="bg-muted/60">
                      {Array.from({ length: columnCount }).map((_, index) => (
                        <th
                          className="border-b border-border px-3 py-2 text-left font-medium text-foreground"
                          key={index}
                          scope="col"
                        >
                          <div className="flex items-center gap-1">
                            {editable ? (
                              <Input
                                aria-label={`表格 ${tableIndex + 1} 表头 ${index + 1}`}
                                className="h-8 min-w-24 border-0 bg-transparent px-1 text-sm font-medium focus-visible:ring-1"
                                value={table.headers[index] ?? `列 ${index + 1}`}
                                onChange={(event) => setHeader(index, event.target.value)}
                              />
                            ) : (
                              <span>{table.headers[index] ?? `列 ${index + 1}`}</span>
                            )}
                            {editable && (
                              <Button
                                aria-label={`删除第 ${index + 1} 列`}
                                type="button"
                                size="icon"
                                variant="ghost"
                                className="size-7 shrink-0"
                                disabled={columnCount <= 1}
                                onClick={() => removeColumn(index)}
                              >
                                <Minus className="size-3" />
                              </Button>
                            )}
                          </div>
                        </th>
                      ))}
                      {editable && (
                        <th className="w-12 border-b border-border px-2 py-2 text-right font-medium text-muted-foreground">
                          操作
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {table.rows.length === 0 ? (
                      <tr>
                        <td
                          className="border-b border-border px-3 py-4 text-sm text-muted-foreground"
                          colSpan={columnCount + (editable ? 1 : 0)}
                        >
                          暂无行。
                        </td>
                      </tr>
                    ) : (
                      table.rows.map((row, rowIndex) => (
                        <tr className="odd:bg-background even:bg-muted/20" key={rowIndex}>
                          {Array.from({ length: columnCount }).map((_, cellIndex) => (
                            <td
                              className={cn(
                                'border-b border-border px-3 py-2 align-top',
                                !editable && 'text-muted-foreground last:border-b-0',
                              )}
                              key={cellIndex}
                            >
                              {editable ? (
                                <Input
                                  aria-label={`表格 ${tableIndex + 1} 第 ${rowIndex + 1} 行第 ${
                                    cellIndex + 1
                                  } 列`}
                                  className="h-8 min-w-28 border-0 bg-transparent px-1 text-sm focus-visible:ring-1"
                                  value={row[cellIndex] ?? ''}
                                  onChange={(event) =>
                                    setCell(rowIndex, cellIndex, event.target.value)
                                  }
                                />
                              ) : (
                                (row[cellIndex] ?? '')
                              )}
                            </td>
                          ))}
                          {editable && (
                            <td className="border-b border-border px-2 py-2 text-right">
                              <Button
                                aria-label={`删除第 ${rowIndex + 1} 行`}
                                type="button"
                                size="icon"
                                variant="ghost"
                                className="size-7"
                                onClick={() => removeRow(rowIndex)}
                              >
                                <Minus className="size-3" />
                              </Button>
                            </td>
                          )}
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
              {editable ? (
                <div className="border-t border-border px-3 py-2">
                  <Input
                    aria-label={`表格 ${tableIndex + 1} 备注`}
                    className="h-8 bg-background text-xs"
                    placeholder="备注"
                    value={table.footnote ?? ''}
                    onChange={(event) => setFootnote(event.target.value)}
                  />
                </div>
              ) : (
                table.footnote && (
                  <p className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
                    {table.footnote}
                  </p>
                )
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
