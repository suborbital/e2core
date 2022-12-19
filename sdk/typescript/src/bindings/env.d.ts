export enum LogLevel {
  Null = 0,
  Error = 1,
  Warn = 2,
  Info = 3,
  Debug = 4,
}
export enum HttpMethod {
  Get = 0,
  Head = 1,
  Options = 2,
  Post = 3,
  Put = 4,
  Patch = 5,
  Delete = 6,
}
export enum FieldType {
  Meta = 0,
  Body = 1,
  Header = 2,
  Params = 3,
  State = 4,
  Query = 5,
}
export enum QueryType {
  Select = 0,
  Insert = 1,
  Update = 2,
  Delete = 3,
}
export class Env {
  
  // The WebAssembly instance that this class is operating with.
  // This is only available after the `instantiate` method has
  // been called.
  instance: WebAssembly.Instance;
  
  // Constructs a new instance with internal state necessary to
  // manage a wasm instance.
  //
  // Note that this does not actually instantiate the WebAssembly
  // instance or module, you'll need to call the `instantiate`
  // method below to "activate" this class.
  constructor();
  
  // This is a low-level method which can be used to add any
  // intrinsics necessary for this instance to operate to an
  // import object.
  //
  // The `import` object given here is expected to be used later
  // to actually instantiate the module this class corresponds to.
  // If the `instantiate` method below actually does the
  // instantiation then there's no need to call this method, but
  // if you're instantiating manually elsewhere then this can be
  // used to prepare the import object for external instantiation.
  addToImports(imports: any): void;
  
  // Initializes this object with the provided WebAssembly
  // module/instance.
  //
  // This is intended to be a flexible method of instantiating
  // and completion of the initialization of this class. This
  // method must be called before interacting with the
  // WebAssembly object.
  //
  // The first argument to this method is where to get the
  // wasm from. This can be a whole bunch of different types,
  // for example:
  //
  // * A precompiled `WebAssembly.Module`
  // * A typed array buffer containing the wasm bytecode.
  // * A `Promise` of a `Response` which is used with
  //   `instantiateStreaming`
  // * A `Response` itself used with `instantiateStreaming`.
  // * An already instantiated `WebAssembly.Instance`
  //
  // If necessary the module is compiled, and if necessary the
  // module is instantiated. Whether or not it's necessary
  // depends on the type of argument provided to
  // instantiation.
  //
  // If instantiation is performed then the `imports` object
  // passed here is the list of imports used to instantiate
  // the instance. This method may add its own intrinsics to
  // this `imports` object too.
  instantiate(
  module: WebAssembly.Module | BufferSource | Promise<Response> | Response | WebAssembly.Instance,
  imports?: any,
  ): Promise<void>;
  returnResult(res: Uint8Array, ident: number): void;
  returnError(code: number, res: string, ident: number): void;
  logMsg(msg: string, level: LogLevel, ident: number): void;
  fetchUrl(method: HttpMethod, url: string, body: Uint8Array, ident: number): number;
  graphqlQuery(endpoint: string, query: string, ident: number): number;
  cacheSet(key: string, value: Uint8Array, ttl: number, ident: number): number;
  cacheGet(key: string, ident: number): number;
  requestGetField(fieldType: FieldType, key: string, ident: number): number;
  getStaticFile(name: string, ident: number): number;
  dbExec(queryType: QueryType, name: string, ident: number): number;
  getFfiResult(ptr: number, ident: number): number;
  addFfiVar(name: string, val: string, ident: number): number;
  returnAbort(msg: string, file: string, lineNum: number, colNum: number, ident: number): void;
}
