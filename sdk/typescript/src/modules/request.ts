import Base from "./base";
import { FieldType } from "../bindings/env";

const decoder = new TextDecoder();

export default class Request extends Base {
  method(): string {
    return decoder.decode(this.getField(FieldType.Meta, "method"));
  }

  url(): string {
    return decoder.decode(this.getField(FieldType.Meta, "url"));
  }

  id(): string {
    return decoder.decode(this.getField(FieldType.Meta, "id"));
  }

  body(): string {
    return decoder.decode(this.getField(FieldType.Body, "body"));
  }

  bodyBytes(): Uint8Array {
    return this.getField(FieldType.Body, "body");
  }

  bodyField(key: string): string {
    return decoder.decode(this.getField(FieldType.Body, key));
  }

  header(key: string): string {
    return decoder.decode(this.getField(FieldType.Header, key));
  }

  urlParam(key: string): string {
    return decoder.decode(this.getField(FieldType.Params, key));
  }

  state(key: string): string {
    return decoder.decode(this.getField(FieldType.State, key));
  }

  stateBytes(key: string): Uint8Array {
    return this.getField(FieldType.State, key);
  }

  private getField(field: FieldType, key: string): Uint8Array {
    const resultSize = this.env.requestGetField(field, key, this.ident);
    return this.ffiResult(resultSize);
  }
}
