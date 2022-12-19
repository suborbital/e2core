import Base from "./base";
import { HttpMethod } from "../bindings/env";
import { renderHeaderString } from "./helpers";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

type Headers = { [key: string]: string };

export class HttpResponse {
  private value: Uint8Array;

  constructor(value: Uint8Array) {
    this.value = value;
  }

  arrayBuffer(): ArrayBuffer {
    // This is safe because `ffiResult` allocates fresh buffers
    return this.value.buffer;
  }

  json(): object {
    return JSON.parse(this.text());
  }

  text(): string {
    return decoder.decode(this.value);
  }
}

export default class Http extends Base {
  get(url: string, headers?: Headers): HttpResponse {
    return this.request(HttpMethod.Get, url, new Uint8Array([]), headers || {});
  }

  head(url: string, headers?: Headers): HttpResponse {
    return this.request(
      HttpMethod.Head,
      url,
      new Uint8Array([]),
      headers || {}
    );
  }

  options(url: string, headers?: Headers): HttpResponse {
    return this.request(
      HttpMethod.Options,
      url,
      new Uint8Array([]),
      headers || {}
    );
  }

  post(
    url: string,
    body: string | Uint8Array,
    headers?: Headers
  ): HttpResponse {
    return this.request(HttpMethod.Post, url, body, headers || {});
  }

  put(url: string, body: string | Uint8Array, headers?: Headers): HttpResponse {
    return this.request(HttpMethod.Put, url, body, headers || {});
  }

  patch(
    url: string,
    body: string | Uint8Array,
    headers?: Headers
  ): HttpResponse {
    return this.request(HttpMethod.Patch, url, body, headers || {});
  }

  delete(url: string, headers?: Headers): HttpResponse {
    return this.request(
      HttpMethod.Delete,
      url,
      new Uint8Array([]),
      headers || {}
    );
  }

  private request(
    method: HttpMethod,
    url: string,
    body: string | Uint8Array,
    headers: Headers
  ): HttpResponse {
    let bodyBytes;
    if (typeof body === "string") {
      bodyBytes = encoder.encode(body);
    } else {
      bodyBytes = body;
    }

    const headerString = renderHeaderString(headers);
    const fullUrl = url + headerString;

    const resultSize = this.env.fetchUrl(
      method,
      fullUrl,
      bodyBytes,
      this.ident
    );
    return new HttpResponse(this.ffiResult(resultSize));
  }
}
