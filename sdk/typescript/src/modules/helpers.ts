export function renderHeaderString(headers: { [key: string]: string }): string {
  const entries = Object.entries(headers);
  const headerPairs = entries.map(([key, value]) => `::${key}:${value}`);
  return headerPairs.join("");
}
