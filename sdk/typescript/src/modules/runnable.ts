import Base from "./base";

const encoder = new TextEncoder();

export default class Runnable extends Base {
  returnResult(result: string | Uint8Array) {
    const bytes = typeof result === "string" ? encoder.encode(result) : result;
    this.env.returnResult(bytes, this.ident);
  }

  returnError(code: number, error: string) {
    this.env.returnError(code, error, this.ident);
  }
}
