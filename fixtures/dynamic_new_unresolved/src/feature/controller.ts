const ctor = serviceMap;

export function handler() {
  return new ctor();
}
