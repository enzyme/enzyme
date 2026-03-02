interface EnzymeRuntimeConfig {
  telemetry?: boolean;
}

interface Window {
  __ENZYME_CONFIG__?: EnzymeRuntimeConfig;
}
