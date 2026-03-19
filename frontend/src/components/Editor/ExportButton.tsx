import { useState, useCallback } from "react";
import { FilePdf } from "@phosphor-icons/react";
import { useLocale } from "@/i18n";
import { exportToPdf } from "@/lib/export-pdf";
import styles from "./ExportButton.module.css";

interface ExportButtonProps {
  getMarkdown: (() => string) | null;
  disabled?: boolean;
}

export function ExportButton({ getMarkdown, disabled }: ExportButtonProps) {
  const { locale, t } = useLocale();
  const [exporting, setExporting] = useState(false);

  const handleExport = useCallback(async () => {
    if (!getMarkdown || exporting) return;

    setExporting(true);
    try {
      const markdown = getMarkdown();
      await exportToPdf(markdown, locale);
    } catch (err) {
      console.error("PDF export failed:", err);
    } finally {
      setExporting(false);
    }
  }, [getMarkdown, locale, exporting]);

  return (
    <button
      type="button"
      className={styles.btn}
      onClick={handleExport}
      disabled={disabled || exporting || !getMarkdown}
      title={exporting ? t.exportButton.exporting : t.exportButton.title}
    >
      <FilePdf size={16} />
    </button>
  );
}
