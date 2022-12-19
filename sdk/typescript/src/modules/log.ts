import Base from "./base";
import { LogLevel } from "../bindings/env";

export default class Log extends Base {
  info(message: string) {
    this.log(message, LogLevel.Info);
  }

  warn(message: string) {
    this.log(message, LogLevel.Warn);
  }

  error(message: string) {
    this.log(message, LogLevel.Error);
  }

  debug(message: string) {
    this.log(message, LogLevel.Debug);
  }

  private log(message: string, level: LogLevel) {
    this.env.logMsg(message, level, this.ident);
  }
}
