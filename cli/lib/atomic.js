/**
 * Atomic file write utilities for concurrent safety.
 *
 * Write-to-tmp + rename pattern prevents partial reads and data loss
 * when multiple mint sessions write to the same file concurrently.
 */
import fs from 'fs';
import path from 'path';
import { randomBytes } from 'crypto';

/**
 * Write a file atomically: write to a temp file in the same directory, then rename.
 * rename() is atomic on POSIX when src and dst are on the same filesystem.
 *
 * @param {string} filePath - target file path
 * @param {string} content - file content to write
 */
export function atomicWriteSync(filePath, content) {
  const dir = path.dirname(filePath);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });

  const tmpPath = path.join(dir, `.${path.basename(filePath)}.${randomBytes(4).toString('hex')}.tmp`);
  try {
    fs.writeFileSync(tmpPath, content);
    fs.renameSync(tmpPath, filePath);
  } catch (err) {
    // Clean up temp file on failure
    try { fs.unlinkSync(tmpPath); } catch { /* ignore */ }
    throw err;
  }
}

/**
 * Write a JSON file atomically with pretty formatting.
 *
 * @param {string} filePath - target file path
 * @param {object} data - JSON-serializable data
 */
export function atomicWriteJsonSync(filePath, data) {
  atomicWriteSync(filePath, JSON.stringify(data, null, 2) + '\n');
}
