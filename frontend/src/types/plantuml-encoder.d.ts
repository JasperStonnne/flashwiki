declare module "plantuml-encoder" {
  export function encode(data: string): string;
  export function decode(encoded: string): string;
}
