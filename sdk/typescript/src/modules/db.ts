import { Env, QueryType } from "../bindings/env";
import Base from "./base";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

type Variables = { [key: string]: string };

export default class DB extends Base {
  select(name: string, variables?: Variables): object {
    return this.exec(QueryType.Select, name, variables || {});
  }

  insert(
    name: string,
    variables?: Variables
  ): { lastInsertID?: number | string } {
    return this.exec(QueryType.Insert, name, variables || {});
  }

  update(name: string, variables?: Variables): { rowsAffected: number } {
    // The host call is guaranteed to return `rowsAffected`
    // @ts-ignore
    return this.exec(QueryType.Update, name, variables || {});
  }

  delete(name: string, variables?: Variables): { rowsAffected: number } {
    // The host call is guaranteed to return `rowsAffected`
    // @ts-ignore
    return this.exec(QueryType.Delete, name, variables || {});
  }

  private exec(
    queryType: QueryType,
    name: string,
    variables: Variables
  ): object {
    Object.entries(variables).forEach(([name, value]) => {
      this.env.addFfiVar(name, value, this.ident);
    });

    const resultSize = this.env.dbExec(queryType, name, this.ident);
    const result = decoder.decode(this.ffiResult(resultSize));

    if (result) {
      return JSON.parse(result);
    } else {
      return {};
    }
  }
}
