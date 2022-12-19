import Base from "./base";
import { renderHeaderString } from "./helpers";

const decoder = new TextDecoder();

export default class GraphQL extends Base {
  query(
    endpoint: string,
    query: string,
    headers?: { [key: string]: string }
  ): string {
    const headerString = headers ? renderHeaderString(headers) : "";
    const url = endpoint + headerString;
    const resultSize = this.env.graphqlQuery(url, query, this.ident);
    const result = this.ffiResult(resultSize);
    return decoder.decode(result);
  }
}
