import { useState, useRef, useEffect, useCallback } from 'react'

const API_BASE = (import.meta.env.BASE_URL ?? '/').replace(/\/$/, '')

export interface UseTTSReturn {
  isPlaying: boolean
  play: (text: string) => Promise<void>
  stop: () => void
}

// A single shared Audio element reused across all TTS playback.
// iOS Safari blocks audio.play() on newly created Audio elements unless
// triggered by a direct user gesture. By reusing one element that was
// "unlocked" during the first user-initiated play, subsequent calls
// (e.g. after an SSE stream completes) can play without gesture.
let sharedAudio: HTMLAudioElement | null = null

function getSharedAudio(): HTMLAudioElement {
  if (!sharedAudio) {
    sharedAudio = new Audio()
  }
  return sharedAudio
}

export function useTTS(): UseTTSReturn {
  const [isPlaying, setIsPlaying] = useState(false)
  const blobUrlRef = useRef<string | null>(null)
  const resolveRef = useRef<(() => void) | null>(null)
  const rejectRef = useRef<((err: Error) => void) | null>(null)

  // Clean up on unmount
  useEffect(() => {
    return () => {
      const audio = getSharedAudio()
      audio.pause()
      audio.removeAttribute('src')
      audio.load()
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current)
        blobUrlRef.current = null
      }
    }
  }, [])

  const stop = useCallback(() => {
    const audio = getSharedAudio()
    audio.pause()
    audio.removeAttribute('src')
    audio.load()
    if (blobUrlRef.current) {
      URL.revokeObjectURL(blobUrlRef.current)
      blobUrlRef.current = null
    }
    setIsPlaying(false)
    // Resolve any pending promise so callers don't hang
    if (resolveRef.current) {
      resolveRef.current()
      resolveRef.current = null
      rejectRef.current = null
    }
  }, [])

  const play = useCallback(
    async (text: string): Promise<void> => {
      stop()

      const token = localStorage.getItem('access_token')
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }

      const res = await fetch(`${API_BASE}/api/v1/speech/tts`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ text, speed: 1.0 }),
      })

      if (!res.ok) {
        throw new Error(`TTS request failed: ${res.status}`)
      }

      const blob = await res.blob()
      const url = URL.createObjectURL(blob)

      // Revoke the previous blob URL if any
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current)
      }
      blobUrlRef.current = url

      const audio = getSharedAudio()

      return new Promise<void>((resolve, reject) => {
        resolveRef.current = resolve
        rejectRef.current = reject

        audio.onended = () => {
          if (blobUrlRef.current) {
            URL.revokeObjectURL(blobUrlRef.current)
            blobUrlRef.current = null
          }
          setIsPlaying(false)
          resolveRef.current = null
          rejectRef.current = null
          resolve()
        }

        audio.onerror = () => {
          if (blobUrlRef.current) {
            URL.revokeObjectURL(blobUrlRef.current)
            blobUrlRef.current = null
          }
          setIsPlaying(false)
          resolveRef.current = null
          rejectRef.current = null
          reject(new Error('Audio playback failed'))
        }

        // Set src and play — reusing the same element preserves the
        // iOS "user gesture unlock" from previous interactions.
        audio.src = url
        setIsPlaying(true)

        audio.play().catch((err) => {
          setIsPlaying(false)
          resolveRef.current = null
          rejectRef.current = null
          reject(err)
        })
      })
    },
    [stop],
  )

  return { isPlaying, play, stop }
}
