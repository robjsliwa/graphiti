/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_GRAPHITI_URL: string;
  readonly VITE_GRAPHITI_TOKEN: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
