import { motion } from 'framer-motion'

const DOT_COUNT = 3
const BOUNCE_DELAY = 0.15 // seconds between each dot's bounce

/**
 * Three bouncing dots that indicate the AI is preparing a response.
 */
export function TypingIndicator() {
  return (
    <div className="flex items-end gap-1 px-4 py-3 bg-ai-bubble rounded-2xl w-fit max-w-[80%]">
      {Array.from({ length: DOT_COUNT }).map((_, i) => (
        <motion.span
          key={i}
          className="block w-2 h-2 rounded-full bg-neutral-400"
          animate={{ y: [0, -6, 0] }}
          transition={{
            duration: 0.6,
            repeat: Infinity,
            delay: i * BOUNCE_DELAY,
            ease: 'easeInOut',
          }}
        />
      ))}
    </div>
  )
}
