/**
 * Chat UI state — sessions metadata, per-session messages, streaming flag, error tracking.
 *
 * QASession does NOT embed messages — they are stored separately in `messagesBySession`.
 * Only session IDs are persisted to localStorage so the sidebar can
 * restore the session list across page reloads.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { QAMessage, QASession, SessionAttachmentSummary } from '@/lib/types'

export type ActiveChatStream = {
  abort: () => void
  assistantMessageId: string
  sessionId: string
}

export type QaUnreadCompletion = {
  completedAt: string
  messageId?: string
  sessionId: string
}

export interface ChatState {
  /** Full session metadata objects (in-memory, fetched from server or created locally). */
  sessions: QASession[]
  /** Session IDs persisted to localStorage for session recovery. */
  sessionIds: string[]
  /** Currently selected session. */
  activeId: string | null
  /** Whether an SSE stream is in progress. */
  streaming: boolean
  /** Volatile abort controller for the currently running QA stream. */
  activeStream: ActiveChatStream | null
  /** Whether the QA chat route is currently visible to the user. */
  qaChatVisible: boolean
  /** Completed answer that finished while the QA chat route was not visible. */
  qaUnreadCompletion: QaUnreadCompletion | null
  /** Last fatal error message for display. */
  error: string | null
  /** The user message that triggered a fatal error (for retry). */
  lastFailedMsg: string | null
  /** Messages keyed by sessionId (QASession does not embed messages). */
  messagesBySession: Record<string, QAMessage[]>
  /** Attachments keyed by sessionId. */
  attachmentsBySession: Record<string, SessionAttachmentSummary[]>
  /** Attachment IDs excluded from the next message send, keyed by sessionId. */
  excludedAttachmentIds: Record<string, string[]>

  // ── Actions ──

  /** Bulk-set session metadata (used when syncing from server). */
  setSessions: (sessions: QASession[]) => void
  setSessionIds: (ids: string[]) => void
  setActiveId: (id: string | null) => void
  /** Prepend a new session metadata, deduping by sessionId. Also updates persisted sessionIds. */
  addSession: (session: QASession) => void
  /** Remove a session, its messages, and its persisted id. Clears activeId if it matches. */
  removeSession: (sessionId: string) => void
  /** Replace the messages array for a given session. */
  updateSessionMessages: (sessionId: string, messages: QAMessage[]) => void
  /** Prepend a new message to a session's message list. */
  appendSessionMessages: (sessionId: string, messages: QAMessage[]) => void
  setStreaming: (streaming: boolean) => void
  setActiveStream: (stream: ActiveChatStream | null) => void
  abortActiveStream: () => void
  setQaChatVisible: (visible: boolean) => void
  markQaCompletionUnread: (completion: QaUnreadCompletion) => void
  clearQaUnreadCompletion: () => void
  setError: (error: string | null) => void
  setLastFailedMsg: (msg: string | null) => void
  /** Reset all state and clear persisted sessionIds from localStorage. */
  reset: () => void
  clearError: () => void
  /** Replace all attachments for a session. */
  setSessionAttachments: (sessionId: string, attachments: SessionAttachmentSummary[]) => void
  /** Add a single attachment to a session (deduped by id). */
  addAttachment: (sessionId: string, attachment: SessionAttachmentSummary) => void
  /** Patch a single attachment by id. */
  updateAttachment: (
    sessionId: string,
    attachmentId: string,
    patch: Partial<SessionAttachmentSummary>,
  ) => void
  /** Remove a single attachment by id. */
  removeAttachment: (sessionId: string, attachmentId: string) => void
  /** Set excluded attachment IDs for a session. */
  setExcludedAttachmentIds: (sessionId: string, ids: string[]) => void
  /** Toggle an attachment's inclusion state for the next send. */
  toggleAttachmentExcluded: (sessionId: string, attachmentId: string) => void
}

export const useChatStore = create<ChatState>()(
  persist(
    (set, get) => ({
      sessions: [],
      sessionIds: [],
      activeId: null,
      streaming: false,
      activeStream: null,
      qaChatVisible: false,
      qaUnreadCompletion: null,
      error: null,
      lastFailedMsg: null,
      messagesBySession: {},
      attachmentsBySession: {},
      excludedAttachmentIds: {},

      setSessions: (sessions) => set({ sessions }),

      setSessionIds: (ids) => set({ sessionIds: ids }),

      setActiveId: (id) => set({ activeId: id }),

      addSession: (session) =>
        set((state) => {
          if (state.sessions.some((s) => s.id === session.id)) {
            return state
          }
          return {
            sessions: [session, ...state.sessions],
            sessionIds: [session.id, ...state.sessionIds.filter((sid) => sid !== session.id)],
          }
        }),

      removeSession: (sessionId) => {
        const activeStream = get().activeStream
        if (activeStream?.sessionId === sessionId) activeStream.abort()
        set((state) => {
          const { [sessionId]: _removedMessages, ...restMessages } = state.messagesBySession
          const { [sessionId]: _removedAttachments, ...restAttachments } =
            state.attachmentsBySession
          const { [sessionId]: _removedExcluded, ...restExcluded } = state.excludedAttachmentIds
          return {
            sessions: state.sessions.filter((s) => s.id !== sessionId),
            sessionIds: state.sessionIds.filter((sid) => sid !== sessionId),
            activeId: state.activeId === sessionId ? null : state.activeId,
            activeStream: state.activeStream?.sessionId === sessionId ? null : state.activeStream,
            messagesBySession: restMessages,
            attachmentsBySession: restAttachments,
            excludedAttachmentIds: restExcluded,
            qaUnreadCompletion:
              state.qaUnreadCompletion?.sessionId === sessionId ? null : state.qaUnreadCompletion,
          }
        })
      },

      updateSessionMessages: (sessionId, messages) =>
        set((state) => ({
          messagesBySession: {
            ...state.messagesBySession,
            [sessionId]: messages,
          },
        })),

      appendSessionMessages: (sessionId, messages) =>
        set((state) => ({
          messagesBySession: {
            ...state.messagesBySession,
            [sessionId]: [...(state.messagesBySession[sessionId] ?? []), ...messages],
          },
        })),

      setStreaming: (streaming) => set({ streaming }),

      setActiveStream: (activeStream) => set({ activeStream }),

      abortActiveStream: () => {
        get().activeStream?.abort()
      },

      setQaChatVisible: (qaChatVisible) => set({ qaChatVisible }),

      markQaCompletionUnread: (qaUnreadCompletion) => set({ qaUnreadCompletion }),

      clearQaUnreadCompletion: () => set({ qaUnreadCompletion: null }),

      setError: (error) => set({ error }),

      setLastFailedMsg: (msg) => set({ lastFailedMsg: msg }),

      reset: () => {
        get().activeStream?.abort()
        set({
          sessions: [],
          sessionIds: [],
          activeId: null,
          streaming: false,
          activeStream: null,
          qaChatVisible: false,
          qaUnreadCompletion: null,
          error: null,
          lastFailedMsg: null,
          messagesBySession: {},
          attachmentsBySession: {},
          excludedAttachmentIds: {},
        })
        // Delete persisted key *after* set so persist middleware
        // doesn't re-create it from the empty sessionIds.
        try {
          localStorage.removeItem('qa-sessions-ids')
        } catch {
          /* noop */
        }
        try {
          useChatStore.persist.clearStorage()
        } catch {
          /* noop */
        }
      },

      clearError: () => set({ error: null, lastFailedMsg: null }),

      setSessionAttachments: (sessionId, attachments) =>
        set((state) => ({
          attachmentsBySession: {
            ...state.attachmentsBySession,
            [sessionId]: attachments,
          },
        })),

      addAttachment: (sessionId, attachment) =>
        set((state) => {
          const existing = state.attachmentsBySession[sessionId] ?? []
          if (existing.some((a) => a.id === attachment.id)) return state
          return {
            attachmentsBySession: {
              ...state.attachmentsBySession,
              [sessionId]: [attachment, ...existing],
            },
          }
        }),

      updateAttachment: (sessionId, attachmentId, patch) =>
        set((state) => {
          const existing = state.attachmentsBySession[sessionId]
          if (!existing) return state
          return {
            attachmentsBySession: {
              ...state.attachmentsBySession,
              [sessionId]: existing.map((a) => (a.id === attachmentId ? { ...a, ...patch } : a)),
            },
          }
        }),

      removeAttachment: (sessionId, attachmentId) =>
        set((state) => {
          const existing = state.attachmentsBySession[sessionId]
          if (!existing) return state
          return {
            attachmentsBySession: {
              ...state.attachmentsBySession,
              [sessionId]: existing.filter((a) => a.id !== attachmentId),
            },
            excludedAttachmentIds: {
              ...state.excludedAttachmentIds,
              [sessionId]: (state.excludedAttachmentIds[sessionId] ?? []).filter(
                (id) => id !== attachmentId,
              ),
            },
          }
        }),

      setExcludedAttachmentIds: (sessionId, ids) =>
        set((state) => ({
          excludedAttachmentIds: {
            ...state.excludedAttachmentIds,
            [sessionId]: ids,
          },
        })),

      toggleAttachmentExcluded: (sessionId, attachmentId) =>
        set((state) => {
          const current = state.excludedAttachmentIds[sessionId] ?? []
          const isExcluded = current.includes(attachmentId)
          return {
            excludedAttachmentIds: {
              ...state.excludedAttachmentIds,
              [sessionId]: isExcluded
                ? current.filter((id) => id !== attachmentId)
                : [...current, attachmentId],
            },
          }
        }),
    }),
    {
      name: 'qa-sessions-ids',
      partialize: (state) => ({ sessionIds: state.sessionIds }),
    },
  ),
)
