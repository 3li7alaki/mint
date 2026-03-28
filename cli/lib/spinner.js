/**
 * TTY-aware spinner — uses @clack/prompts spinner when interactive,
 * falls back to plain log lines when piped/captured (non-TTY).
 *
 * Prevents spinner animation frames from bloating captured output.
 */
import * as p from '@clack/prompts';

const isTTY = process.stdout.isTTY;

/**
 * Create a spinner that degrades gracefully in non-TTY environments.
 * Same API as @clack/prompts spinner: .start(msg) and .stop(msg).
 */
export function spinner() {
  if (isTTY) return p.spinner();

  let startMsg = '';
  return {
    start(msg = '') {
      startMsg = msg;
      if (msg) console.log(`  ${msg}`);
    },
    stop(msg = '') {
      if (msg) console.log(`  ${msg}`);
    },
    message(msg = '') {
      // no-op in non-TTY — avoids animation frame spam
    },
  };
}
