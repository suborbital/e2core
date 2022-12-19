import Base from "./base";

const decoder = new TextDecoder();

export default class File extends Base {
  getStatic(key: string): string {
    return decoder.decode(this.getStaticBytes(key));
  }

  getStaticBytes(name: string): Uint8Array {
    const resultSize = this.env.getStaticFile(name, this.ident);
    return this.ffiResult(resultSize);
  }
}
