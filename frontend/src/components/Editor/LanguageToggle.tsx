import { useLocale } from "@/i18n";
import styles from "./LanguageToggle.module.css";

export function LanguageToggle() {
  const { locale, setLocale, t } = useLocale();

  return (
    <button
      type="button"
      className={styles.btn}
      onClick={() => setLocale(locale === "en" ? "zh" : "en")}
      title={locale === "en" ? "Switch to Chinese" : "\u5207\u6362\u4E3A\u82F1\u6587"}
    >
      {t.langToggle.label}
    </button>
  );
}
