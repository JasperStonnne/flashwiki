import { createContext, useContext, useState, useCallback, useMemo } from "react";
import type { ReactNode } from "react";
import type { Locale, Dictionary } from "./types";
import { en } from "./en";
import { zh } from "./zh";

export const dictionaries: Record<Locale, Dictionary> = { en, zh };

interface LocaleContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: Dictionary;
}

const LocaleContext = createContext<LocaleContextValue>({
  locale: "zh",
  setLocale: () => {},
  t: zh,
});

interface LocaleProviderProps {
  locale?: Locale;
  onLocaleChange?: (locale: Locale) => void;
  children: ReactNode;
}

export function LocaleProvider({ locale: controlledLocale, onLocaleChange, children }: LocaleProviderProps) {
  const [internalLocale, setInternalLocale] = useState<Locale>(controlledLocale ?? "zh");

  const isControlled = controlledLocale !== undefined;
  const locale = isControlled ? controlledLocale : internalLocale;

  const setLocale = useCallback(
    (newLocale: Locale) => {
      if (!isControlled) {
        setInternalLocale(newLocale);
      }
      onLocaleChange?.(newLocale);
    },
    [isControlled, onLocaleChange]
  );

  const value = useMemo<LocaleContextValue>(
    () => ({ locale, setLocale, t: dictionaries[locale] }),
    [locale, setLocale]
  );

  return <LocaleContext value={value}>{children}</LocaleContext>;
}

export function useLocale(): LocaleContextValue {
  return useContext(LocaleContext);
}
