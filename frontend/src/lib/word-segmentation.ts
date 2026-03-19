/**
 * Word segmentation utilities using Intl.Segmenter.
 *
 * Supports Chinese, Japanese, and other languages with proper
 * word boundary detection.
 */

// Type declarations for Intl.Segmenter (not yet in all TypeScript lib versions)
interface SegmenterSegment {
  segment: string;
  index: number;
  isWordLike?: boolean;
}

interface SegmenterIterator {
  [Symbol.iterator](): IterableIterator<SegmenterSegment>;
}

interface IntlSegmenter {
  segment(input: string): SegmenterIterator;
}

interface IntlSegmenterConstructor {
  new (
    locales?: string | string[],
    options?: { granularity?: "grapheme" | "word" | "sentence" }
  ): IntlSegmenter;
}

declare global {
  interface IntlExtended {
    Segmenter?: IntlSegmenterConstructor;
  }
}

const IntlExt = Intl as unknown as IntlExtended;

export interface WordBoundary {
  start: number;
  end: number;
  word: string;
}

/**
 * Find the word at a given cursor position in text.
 *
 * Uses Intl.Segmenter for proper word boundary detection,
 * which handles Chinese, Japanese, and other CJK languages correctly.
 *
 * @param text The full text content
 * @param cursorPos The cursor position (0-indexed)
 * @returns The word boundary if cursor is within a word, null otherwise
 */
export function findWordAtPosition(
  text: string,
  cursorPos: number
): WordBoundary | null {
  if (!text || cursorPos < 0 || cursorPos > text.length) {
    return null;
  }

  // Use Intl.Segmenter for proper word segmentation
  // Falls back to simple whitespace splitting if not supported
  if (IntlExt.Segmenter) {
    const segmenter = new IntlExt.Segmenter(undefined, { granularity: "word" });
    const segments = segmenter.segment(text);

    let currentPos = 0;
    for (const segment of segments) {
      const segmentEnd = currentPos + segment.segment.length;

      // Check if cursor is within this segment
      if (cursorPos >= currentPos && cursorPos <= segmentEnd) {
        // Only return if it's a word (not whitespace/punctuation)
        if (segment.isWordLike) {
          return {
            start: currentPos,
            end: segmentEnd,
            word: segment.segment,
          };
        }
        return null;
      }

      currentPos = segmentEnd;
    }
  } else {
    // Fallback: simple word boundary detection for ASCII
    // Find word boundaries using regex
    const wordRegex = /[\w\u4e00-\u9fff\u3400-\u4dbf]+/g;
    let match;

    while ((match = wordRegex.exec(text)) !== null) {
      const start = match.index;
      const end = start + match[0].length;

      if (cursorPos >= start && cursorPos <= end) {
        return {
          start,
          end,
          word: match[0],
        };
      }
    }
  }

  return null;
}

/**
 * Get all words in a text string.
 *
 * @param text The text to segment
 * @returns Array of word segments
 */
export function getWords(text: string): WordBoundary[] {
  const words: WordBoundary[] = [];

  if (!text) {
    return words;
  }

  if (IntlExt.Segmenter) {
    const segmenter = new IntlExt.Segmenter(undefined, { granularity: "word" });
    const segments = segmenter.segment(text);

    let currentPos = 0;
    for (const segment of segments) {
      if (segment.isWordLike) {
        words.push({
          start: currentPos,
          end: currentPos + segment.segment.length,
          word: segment.segment,
        });
      }
      currentPos += segment.segment.length;
    }
  } else {
    // Fallback
    const wordRegex = /[\w\u4e00-\u9fff\u3400-\u4dbf]+/g;
    let match;

    while ((match = wordRegex.exec(text)) !== null) {
      words.push({
        start: match.index,
        end: match.index + match[0].length,
        word: match[0],
      });
    }
  }

  return words;
}
