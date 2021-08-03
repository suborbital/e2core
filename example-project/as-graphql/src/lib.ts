import { graphQLQuery, logInfo } from "@suborbital/suborbital"

export function run(_: ArrayBuffer): ArrayBuffer {
	let result = graphQLQuery("https://api.github.com/graphql", "{ repository (owner: \"suborbital\", name: \"reactr\") { name, nameWithOwner }}")
	if (result.byteLength == 0) {
		return String.UTF8.encode("failed")
	}

	logInfo(String.UTF8.decode(result))

	return result
}