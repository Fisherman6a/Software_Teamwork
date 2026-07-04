import { QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'

import type { DocumentSummary } from '@/api/knowledge'
import { createTestQueryClient } from '@/test/render'

import { documentKeys, useUploadDocument, useUploadDocumentBatch } from './use-documents'

function createDocument(overrides: Partial<DocumentSummary> = {}): DocumentSummary {
  return {
    chunkCount: 0,
    contentType: 'application/pdf',
    createdAt: '2026-07-01T00:00:00.000Z',
    createdBy: 'user-1',
    errorCode: null,
    errorMessage: null,
    id: 'doc-1',
    jobId: null,
    knowledgeBaseId: 'kb-1',
    name: 'Manual.PDF',
    parserBackend: null,
    sizeBytes: 2048,
    status: 'uploaded',
    tags: [],
    updatedAt: '2026-07-01T00:00:00.000Z',
    ...overrides,
  }
}

describe('useUploadDocument', () => {
  it('invalidates document lists and knowledge base summaries after a successful upload', async () => {
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries').mockResolvedValue()

    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(JSON.stringify({ data: createDocument(), requestId: 'req-upload' }), {
            headers: { 'Content-Type': 'application/json' },
            status: 201,
          }),
      ),
    )

    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => useUploadDocument(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({
        file: new File(['manual'], 'Manual.PDF', { type: 'application/pdf' }),
        knowledgeBaseId: 'kb-1',
        tags: ['规程'],
      })
    })

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: documentKeys.lists() })
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['knowledge-bases'] })
    })
  })
})

describe('useUploadDocumentBatch', () => {
  it('invalidates document lists and knowledge bases only when at least one file succeeds', async () => {
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries').mockResolvedValue()

    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              data: {
                failedCount: 1,
                results: [
                  {
                    error: { code: 'validation_error', message: 'file must not be empty' },
                    filename: 'empty.txt',
                    status: 'failed',
                  },
                ],
                successCount: 0,
                totalCount: 1,
              },
              requestId: 'req-batch-failed',
            }),
            {
              headers: { 'Content-Type': 'application/json' },
              status: 207,
            },
          ),
        )
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              data: {
                failedCount: 1,
                results: [
                  {
                    document: createDocument({ id: 'doc-1', name: 'Manual.PDF' }),
                    filename: 'Manual.PDF',
                    status: 'uploaded',
                  },
                  {
                    error: { code: 'validation_error', message: 'file must not be empty' },
                    filename: 'empty.txt',
                    status: 'failed',
                  },
                ],
                successCount: 1,
                totalCount: 2,
              },
              requestId: 'req-batch-partial',
            }),
            {
              headers: { 'Content-Type': 'application/json' },
              status: 207,
            },
          ),
        ),
    )

    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => useUploadDocumentBatch(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({
        files: [new File([], 'empty.txt', { type: 'text/plain' })],
        knowledgeBaseId: 'kb-1',
      })
    })

    expect(invalidateSpy).not.toHaveBeenCalled()

    await act(async () => {
      await result.current.mutateAsync({
        files: [
          new File(['manual'], 'Manual.PDF', { type: 'application/pdf' }),
          new File([], 'empty.txt', { type: 'text/plain' }),
        ],
        knowledgeBaseId: 'kb-1',
      })
    })

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: documentKeys.lists() })
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['knowledge-bases'] })
    })
  })
})
