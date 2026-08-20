export function isCloudModel(model: string): boolean {
  const separator = model.lastIndexOf(":");
  if (separator < 0) return false;

  const suffix = model.slice(separator + 1).trim().toLowerCase();
  return (
    suffix === "cloud" ||
    (!suffix.includes("/") && suffix.endsWith("-cloud"))
  );
}
