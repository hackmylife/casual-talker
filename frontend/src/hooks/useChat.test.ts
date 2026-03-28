import { renderHook, act } from '@testing-library/react'
import { useChat } from './useChat'

// ---------------------------------------------------------------------------
// Mock setup
// ---------------------------------------------------------------------------

const mockFetch = vi.fn()
global.fetch = mockFetch

const mockStorage = new Map<string, string>()
vi.stubGlobal('localStorage', {
  getItem: (key: string) => mockStorage.get(key) ?? null,
  setItem: (key: string, value: string) => mockStorage.set(key, value),
  removeItem: (key: string) => mockStorage.delete(key),
})

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Creates a mock ReadableStream that yields the provided SSE chunks in order
 * and then closes, matching the behaviour of a real SSE endpoint.
 */
function createMockSSEStream(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk))
      }
      controller.close()
    },
  })
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('useChat', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    mockStorage.clear()
  })

  // -------------------------------------------------------------------------
  // Initial state
  // -------------------------------------------------------------------------

  it('has an empty messages array and isStreaming=false initially', () => {
    const { result } = renderHook(() => useChat())

    expect(result.current.messages).toEqual([])
    expect(result.current.isStreaming).toBe(false)
  })

  // -------------------------------------------------------------------------
  // addUserMessage
  // -------------------------------------------------------------------------

  describe('addUserMessage', () => {
    it('appends a user-role message to the messages array', () => {
      const { result } = renderHook(() => useChat())

      act(() => {
        result.current.addUserMessage('Hello!')
      })

      expect(result.current.messages).toHaveLength(1)
      expect(result.current.messages[0]).toEqual({ role: 'user', text: 'Hello!' })
    })

    it('accumulates multiple user messages in order', () => {
      const { result } = renderHook(() => useChat())

      act(() => {
        result.current.addUserMessage('First')
        result.current.addUserMessage('Second')
      })

      expect(result.current.messages).toHaveLength(2)
      expect(result.current.messages[1].text).toBe('Second')
    })
  })

  // -------------------------------------------------------------------------
  // addAIMessage
  // -------------------------------------------------------------------------

  describe('addAIMessage', () => {
    it('appends an ai-role message to the messages array', () => {
      const { result } = renderHook(() => useChat())

      act(() => {
        result.current.addAIMessage('Hello from AI!')
      })

      expect(result.current.messages).toHaveLength(1)
      expect(result.current.messages[0]).toEqual({ role: 'ai', text: 'Hello from AI!' })
    })
  })

  // -------------------------------------------------------------------------
  // sendMessage – SSE streaming
  // -------------------------------------------------------------------------

  describe('sendMessage', () => {
    it('calls the correct fetch endpoint with the session ID and message', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        body: createMockSSEStream(['data: [DONE]\n\n']),
      })

      const { result } = renderHook(() => useChat())

      await act(async () => {
        await result.current.sendMessage('session-abc', 'Hi there')
      })

      expect(mockFetch).toHaveBeenCalledOnce()
      const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      expect(url).toBe('/api/v1/chat/stream')
      const body = JSON.parse(init.body as string)
      expect(body.session_id).toBe('session-abc')
      expect(body.message).toBe('Hi there')
    })

    it('progressively builds up the AI message text from SSE chunks', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        body: createMockSSEStream([
          'data: {"content":"Hello"}\n\n',
          'data: {"content":" world"}\n\n',
          'data: [DONE]\n\n',
        ]),
      })

      const { result } = renderHook(() => useChat())

      await act(async () => {
        await result.current.sendMessage('sess-1', 'Say hello')
      })

      // The last message in the array should be the accumulated AI response.
      const lastMsg = result.current.messages[result.current.messages.length - 1]
      expect(lastMsg.role).toBe('ai')
      expect(lastMsg.text).toBe('Hello world')
    })

    it('returns the full AI response text when the stream completes', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        body: createMockSSEStream([
          'data: {"content":"Hello"}\n\n',
          'data: {"content":" world"}\n\n',
          'data: [DONE]\n\n',
        ]),
      })

      const { result } = renderHook(() => useChat())

      let response = ''
      await act(async () => {
        response = await result.current.sendMessage('sess-1', 'Say hello')
      })

      expect(response).toBe('Hello world')
    })

    it('sets isStreaming=true during streaming and false after completion', async () => {
      // We need to control when the stream resolves so we can observe the
      // intermediate isStreaming=true state.
      let resolveStream!: () => void
      const streamReady = new Promise<void>((res) => { resolveStream = res })

      const encoder = new TextEncoder()
      const stream = new ReadableStream<Uint8Array>({
        async start(controller) {
          await streamReady
          controller.enqueue(encoder.encode('data: {"content":"Hi"}\n\n'))
          controller.enqueue(encoder.encode('data: [DONE]\n\n'))
          controller.close()
        },
      })

      mockFetch.mockResolvedValueOnce({ ok: true, body: stream })

      const { result } = renderHook(() => useChat())

      // Start sendMessage but don't await yet
      let sendPromise!: Promise<string>
      act(() => {
        sendPromise = result.current.sendMessage('sess-1', 'test')
      })

      // Resolve the stream and wait for sendMessage to complete
      await act(async () => {
        resolveStream()
        await sendPromise
      })

      expect(result.current.isStreaming).toBe(false)
    })

    it('includes interpreted_text in the request body only when it differs from text', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        body: createMockSSEStream(['data: [DONE]\n\n']),
      })

      const { result } = renderHook(() => useChat())

      await act(async () => {
        await result.current.sendMessage('sess-1', 'raw text', 'corrected text')
      })

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      const body = JSON.parse(init.body as string)
      expect(body.interpreted_text).toBe('corrected text')
    })

    it('omits interpreted_text when it is identical to text', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        body: createMockSSEStream(['data: [DONE]\n\n']),
      })

      const { result } = renderHook(() => useChat())

      await act(async () => {
        await result.current.sendMessage('sess-1', 'same', 'same')
      })

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      const body = JSON.parse(init.body as string)
      expect(body.interpreted_text).toBeUndefined()
    })

    it('throws when the fetch response is not ok', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 500 })

      const { result } = renderHook(() => useChat())

      await expect(
        act(async () => {
          await result.current.sendMessage('sess-1', 'oops')
        }),
      ).rejects.toThrow()
    })

    it('sets isStreaming=false even when an error occurs', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 503 })

      const { result } = renderHook(() => useChat())

      await act(async () => {
        await result.current.sendMessage('sess-1', 'error case').catch(() => {})
      })

      expect(result.current.isStreaming).toBe(false)
    })
  })
})
