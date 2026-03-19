import type { Locale } from "@/i18n";
import { PDF_API_URL } from "./pdf-config";

export async function exportToPdf(
  markdown: string,
  locale: Locale,
): Promise<void> {
  const response = await fetch(`${PDF_API_URL}/api/pdf`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ markdown, locale }),
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `PDF export failed (${response.status})`);
  }

  const blob = await response.blob();

  // Extract filename from Content-Disposition header
  const disposition = response.headers.get("Content-Disposition");
  let filename = "Markdown Export.pdf";
  if (disposition) {
    const match = disposition.match(/filename\*?=(?:UTF-8''|"?)([^";]+)/i);
    if (match?.[1]) {
      filename = decodeURIComponent(match[1].replace(/"/g, ""));
    }
  }

  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
