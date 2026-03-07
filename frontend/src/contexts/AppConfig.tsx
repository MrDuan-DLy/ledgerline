import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { getConfig } from '../api/client';

export interface AppConfig {
  currencySymbol: string;
  locale: string;
  supportedFormats: string[];
  appName: string;
}

const defaultConfig: AppConfig = {
  currencySymbol: '£',
  locale: 'en-GB',
  supportedFormats: ['pdf', 'csv'],
  appName: 'Ledgerline',
};

const AppConfigContext = createContext<AppConfig>(defaultConfig);

export function AppConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<AppConfig>(defaultConfig);

  useEffect(() => {
    getConfig()
      .then((data) =>
        setConfig({
          currencySymbol: data.currency_symbol ?? defaultConfig.currencySymbol,
          locale: data.locale ?? defaultConfig.locale,
          supportedFormats: data.supported_formats ?? defaultConfig.supportedFormats,
          appName: data.app_name ?? defaultConfig.appName,
        }),
      )
      .catch(() => {
        // API unavailable — keep defaults
      });
  }, []);

  return (
    <AppConfigContext.Provider value={config}>
      {children}
    </AppConfigContext.Provider>
  );
}

export function useAppConfig(): AppConfig {
  return useContext(AppConfigContext);
}
