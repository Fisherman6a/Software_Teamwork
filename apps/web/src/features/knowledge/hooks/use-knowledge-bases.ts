/**
 * React Query hooks for knowledge base CRUD.
 *
 * Server state managed by TanStack Query with client-side
 * keyword/docType filtering and pagination.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import {
  createKnowledgeBase,
  deleteKnowledgeBase,
  getKnowledgeBase,
  listKnowledgeBases,
  updateKnowledgeBase,
} from '@/api/admin'
import type { UpdateKnowledgeBaseRequest } from '@/lib/types'

// ── Query keys ──

export const knowledgeBaseKeys = {
  all: ['knowledge-bases'] as const,
  lists: () => [...knowledgeBaseKeys.all, 'list'] as const,
  list: (page: number, pageSize: number, keyword?: string, docType?: string) =>
    [...knowledgeBaseKeys.lists(), { page, pageSize, keyword, docType }] as const,
  details: () => [...knowledgeBaseKeys.all, 'detail'] as const,
  detail: (id: string) => [...knowledgeBaseKeys.details(), id] as const,
}

// ── Queries ──

/**
 * Paginated knowledge base list with client-side keyword / docType filtering.
 *
 * The backend list endpoint does not support search params, so all KBs are
 * fetched and filtered/paginated client-side for better UX.
 */
export function useKnowledgeBases(
  page = 1,
  pageSize = 10,
  keyword?: string,
  docType?: string,
) {
  return useQuery({
    queryKey: knowledgeBaseKeys.list(page, pageSize, keyword, docType),
    queryFn: async () => {
      // Fetch a large batch — KB counts are expected to be moderate
      const result = await listKnowledgeBases({ page: 1, pageSize: 200 })
      let items = result.items

      // Client-side keyword filter (name and description)
      if (keyword) {
        const kw = keyword.toLowerCase()
        items = items.filter(
          (kb) =>
            kb.name.toLowerCase().includes(kw) ||
            (kb.description && kb.description.toLowerCase().includes(kw)),
        )
      }

      // Client-side docType filter
      if (docType) {
        items = items.filter((kb) => kb.docType === docType)
      }

      // Client-side pagination
      const total = items.length
      const start = (page - 1) * pageSize
      const paged = items.slice(start, start + pageSize)

      return { items: paged, page: { page, pageSize, total } }
    },
    placeholderData: (prev) => prev,
  })
}

/** Single knowledge base detail. */
export function useKnowledgeBase(id: string) {
  return useQuery({
    queryKey: knowledgeBaseKeys.detail(id),
    queryFn: () => getKnowledgeBase(id),
    enabled: id.length > 0,
  })
}

// ── Mutations ──

/** Create a new knowledge base. */
export function useCreateKnowledgeBase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createKnowledgeBase,
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: knowledgeBaseKeys.lists(),
      })
    },
  })
}

/** Update an existing knowledge base. */
export function useUpdateKnowledgeBase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      ...params
    }: { id: string } & UpdateKnowledgeBaseRequest) =>
      updateKnowledgeBase(id, params),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: knowledgeBaseKeys.lists(),
      })
      void queryClient.invalidateQueries({
        queryKey: knowledgeBaseKeys.detail(variables.id),
      })
    },
  })
}

/** Delete a knowledge base. */
export function useDeleteKnowledgeBase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteKnowledgeBase(id),
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({
        queryKey: knowledgeBaseKeys.lists(),
      })
      queryClient.removeQueries({
        queryKey: knowledgeBaseKeys.detail(id),
      })
    },
  })
}
