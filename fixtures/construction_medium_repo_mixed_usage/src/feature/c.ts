declare const registry: Record<string, new () => unknown>;
declare const key: string;

export function makeDynamic() {
  return new registry[key]();
}
