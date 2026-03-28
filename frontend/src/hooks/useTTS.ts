import { useState, useRef, useEffect, useCallback } from 'react'

const API_BASE = (import.meta.env.BASE_URL ?? '/').replace(/\/$/, '')

export interface UseTTSReturn {
  isPlaying: boolean
  play: (text: string) => Promise<void>
  stop: () => void
}

export function useTTS(): UseTTSReturn {
  const [isPlaying, setIsPlaying] = useState(false)

  const audioRef = useRef<HTMLAudioElement | null>(null)
  const blobUrlRef = useRef<string | null>(null)

  // Clean up audio resources when the component unmounts
  useEffect(() => {
    return () => {
      if (audioRef.current) {
        audioRef.current.pause()
        audioRef.current = null
      }
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current)
        blobUrlRef.current = null
      }
    }
  }, [])

  const stop = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.currentTime = 0
    }
    if (blobUrlRef.current) {
      URL.revokeObjectURL(blobUrlRef.current)
      blobUrlRef.current = null
    }
    setIsPlaying(false)
  }, [])

  const play = useCallback(
    async (text: string): Promise<void> => {
      // Stop any currently playing audio before starting a new one
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
      blobUrlRef.current = url

      return new Promise<void>((resolve, reject) => {
        const audio = new Audio(url)
        audioRef.current = audio

        setIsPlaying(true)

        audio.onended = () => {
          URL.revokeObjectURL(url)
          blobUrlRef.current = null
          audioRef.current = null
          setIsPlaying(false)
          resolve()
        }

        audio.onerror = () => {
          URL.revokeObjectURL(url)
          blobUrlRef.current = null
          audioRef.current = null
          setIsPlaying(false)
          reject(new Error('Audio playback failed'))
        }

        audio.play().catch((err) => {
          setIsPlaying(false)
          reject(err)
        })
      })
    },
    [stop],
  )

  return { isPlaying, play, stop }
}
