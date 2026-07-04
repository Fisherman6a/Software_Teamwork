import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render'

import { KnowledgeBaseMultiSelect } from './knowledge-base-multi-select'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

function knowledgeBase(id: string, name: string) {
  return {
    createdAt: '2026-07-04T00:00:00Z',
    documentCount: 0,
    id,
    name,
    retrievalStrategy: { mode: 'semantic' },
    status: 'ready',
    updatedAt: '2026-07-04T00:00:00Z',
  }
}

function ControlledSelector({ onValueChange }: { onValueChange: (value: string[]) => void }) {
  const [value, setValue] = useState<string[]>([])

  return (
    <KnowledgeBaseMultiSelect
      value={value}
      onChange={(nextValue) => {
        setValue(nextValue)
        onValueChange(nextValue)
      }}
    />
  )
}

describe('KnowledgeBaseMultiSelect', () => {
  it('can page beyond the first 100 knowledge bases and select later results', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    const requestedPages: string[] = []
    const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      requestedPages.push(`${url.searchParams.get('page')}:${url.searchParams.get('pageSize')}`)

      if (url.searchParams.get('page') === '2') {
        return jsonResponse({
          data: [knowledgeBase('kb-101', '第 101 个知识库')],
          page: { page: 2, pageSize: 100, total: 101 },
          requestId: 'req-kb-page-2',
        })
      }

      return jsonResponse({
        data: [knowledgeBase('kb-001', '第 1 个知识库')],
        page: { page: 1, pageSize: 100, total: 101 },
        requestId: 'req-kb-page-1',
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ControlledSelector onValueChange={(value) => values.push(value)} />)

    expect(await screen.findByRole('button', { name: /第 1 个知识库/ })).toBeVisible()
    await user.click(screen.getByRole('button', { name: '下一页知识库' }))

    const secondPageKnowledgeBase = await screen.findByRole('button', { name: /第 101 个知识库/ })
    await user.click(secondPageKnowledgeBase)

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-101']))
    expect(requestedPages).toContain('1:100')
    expect(requestedPages).toContain('2:100')
  })

  it('adds a knowledge base selected from the loaded Gateway list', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse({
          data: [
            knowledgeBase('kb-001', '第 1 个知识库'),
            knowledgeBase('kb-002', '第 2 个知识库'),
          ],
          page: { page: 1, pageSize: 100, total: 101 },
          requestId: 'req-kb-page-1',
        }),
      ),
    )

    renderWithProviders(<ControlledSelector onValueChange={(value) => values.push(value)} />)

    await screen.findByRole('button', { name: /第 1 个知识库/ })
    await user.click(screen.getByRole('combobox', { name: '知识库范围选择' }))
    await user.click(screen.getByRole('option', { name: /第 2 个知识库 \(kb-002\)/ }))
    await user.click(screen.getByRole('button', { name: '添加' }))

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-002']))
    expect(screen.getByTitle('kb-002')).toHaveTextContent('第 2 个知识库')
    expect(screen.queryByLabelText('知识库范围ID')).not.toBeInTheDocument()
  })

  it('keeps manual id fallback for diagnostic retrieval', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse(
          {
            error: {
              code: 'dependency_error',
              message: 'knowledge service unavailable',
              requestId: 'req-kb-failed',
            },
          },
          { status: 502 },
        ),
      ),
    )

    renderWithProviders(<ControlledSelector onValueChange={(value) => values.push(value)} />)

    expect(await screen.findByText('知识库列表依赖失败')).toBeVisible()
    await user.type(screen.getByLabelText('知识库范围手动ID'), 'kb-known, kb-off-page')
    await user.click(screen.getByRole('button', { name: '添加ID' }))

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-known', 'kb-off-page']))
    expect(screen.getByTitle('kb-known')).toBeVisible()
    expect(screen.getByTitle('kb-off-page')).toBeVisible()
  })

  it('keeps the search box editable when the knowledge base list fails', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse(
          {
            error: {
              code: 'dependency_error',
              message: 'knowledge service unavailable',
              requestId: 'req-kb-failed',
            },
          },
          { status: 502 },
        ),
      ),
    )

    renderWithProviders(<ControlledSelector onValueChange={vi.fn()} />)

    expect(await screen.findByText('知识库列表依赖失败')).toBeVisible()
    const searchInput = screen.getByLabelText('知识库范围搜索')
    expect(searchInput).not.toBeDisabled()

    await user.type(searchInput, '运行规程')

    expect(searchInput).toHaveValue('运行规程')
  })
})
