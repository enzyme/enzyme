/// <reference types="nativewind/types" />

// Hermes runtime provides Web Crypto API
declare const crypto: {
  getRandomValues<T extends ArrayBufferView>(array: T): T;
  randomUUID(): `${string}-${string}-${string}-${string}-${string}`;
};
