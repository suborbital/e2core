import { log } from "@suborbital/runnable";

export const run = (input) => {
  let message = "Hello, " + input;

  log.info(message);

  return message;
};
