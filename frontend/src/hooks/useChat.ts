import { useState, useCallback, type Dispatch, type SetStateAction } from 'react'

export interface Message {
  role: 'ai' | 'user'
  text: string
  /** AI's interpretation of what the user likely meant (pronunciation correction) */
  interpretedText?: string
  /** True when interpretedText differs from the raw text */
  isInterpreted?: boolean
}

export interface UseChatReturn {
  messages: Message[]
  isStreaming: boolean
  sendMessage: (sessionId: string, text: string, interpretedText?: string) => Promise<string>
  addUserMessage: (text: string) => void
  addAIMessage: (text: string) => void
  setMessages: Dispatch<SetStateAction<Message[]>>
}

export function useChat(): UseChatReturn {
  const [messages, setMessages] = useState<Message[]>([])
  const [isStreaming, setIsStreaming] = useState(false)

  const addUserMessage = useCallback((text: string) => {
    setMessages((prev) => [...prev, { role: 'user', text }])
  }, [])

  const addAIMessage = useCallback((text: string) => {
    setMessages((prev) => [...prev, { role: 'ai', text }])
  }, [])

  /**
   * Send a message to the chat stream endpoint and progressively update the
   * last AI message in state as SSE chunks arrive.
   * Returns the full AI response text when the stream completes.
   *
   * @param interpretedText - Optional pronunciation-corrected version of text.
   *   When provided (and different from text), the backend uses this for the AI
   *   model so it can respond to the intended meaning.
   */
  const sendMessage = useCallback(
    async (sessionId: string, text: string, interpretedText?: string): Promise<string> => {
      const token = localStorage.getItem('access_token')
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      }
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }

      setIsStreaming(true)

      // Insert a placeholder AI message that will be updated incrementally
      setMessages((prev) => [...prev, { role: 'ai', text: '' }])

      let fullResponse = ''

      try {
        const res = await fetch('/api/v1/chat/stream', {
          method: 'POST',
          headers,
          body: JSON.stringify({
            session_id: sessionId,
            message: text,
            // Only include interpreted_text when it differs from the raw text
            ...(interpretedText && interpretedText !== text
              ? { interpreted_text: interpretedText }
              : {}),
          }),
        })

        if (!res.ok) {
          throw new Error(`Chat stream request failed: ${res.status}`)
        }

        if (!res.body) {
          throw new Error('Response body is null')
        }

        const reader = res.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })

          // Process complete SSE lines from the buffer
          const lines = buffer.split('\n')
          // Keep the last potentially incomplete line in the buffer
          buffer = lines.pop() ?? ''

          for (const line of lines) {
            const trimmed = line.trim()
            if (!trimmed.startsWith('data:')) continue

            const payload = trimmed.slice(5).trim()

            if (payload === '[DONE]') {
              break
            }

            try {
              const parsed = JSON.parse(payload) as { content?: string }
              if (parsed.content) {
                fullResponse += parsed.content
                // Update the last message (the AI placeholder) incrementally
                setMessages((prev) => {
                  const updated = [...prev]
                  updated[updated.length - 1] = {
                    role: 'ai',
                    text: fullResponse,
                  }
                  return updated
                })
              }
            } catch {
              // Ignore malformed JSON chunks
            }
          }
        }
      } finally {
        setIsStreaming(false)
      }

      return fullResponse
    },
    [],
  )

  return {
    messages,
    isStreaming,
    sendMessage,
    addUserMessage,
    addAIMessage,
    setMessages,
  }
}
