interface EnzymeRuntimeConfig {
  telemetry?: {
    enabled: boolean;
    endpoint?: string;
  };
}

interface Window {
  __ENZYME_CONFIG__?: EnzymeRuntimeConfig;
}
