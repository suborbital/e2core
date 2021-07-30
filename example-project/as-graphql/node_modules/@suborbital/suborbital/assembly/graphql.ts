import { graphql_query } from "./env";
import { toFFI, ffi_result, getIdent } from "./ffi";

// send a GraphQL query to the provided endpoint
export function graphQLQuery(endpoint: string, query: string): ArrayBuffer {
	let endpointFFI = toFFI(String.UTF8.encode(endpoint))
	let queryFFI = toFFI(String.UTF8.encode(query))

	let result_size = graphql_query(endpointFFI.ptr, endpointFFI.size, queryFFI.ptr, queryFFI.size, getIdent())

	let result = ffi_result(result_size)

	return result
}