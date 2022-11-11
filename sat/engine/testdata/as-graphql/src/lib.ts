import { graphQLQuery, logInfo } from "@suborbital/suborbital"

export function run(_: ArrayBuffer): ArrayBuffer {
	let result = graphQLQuery("https://api.github.com/graphql", "{ repository (owner: \"suborbital\", name: \"reactr\") { name, nameWithOwner }}")
	let err = result.Err
	if (err) {
		return String.UTF8.encode(err.toString())
	}

	logInfo(String.UTF8.decode(result.Result))

	return result.Result
}