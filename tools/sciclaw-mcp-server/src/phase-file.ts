export function phaseOnePath(sourcePath: string): string {
  const prefix = "phase_0/";
  if (
    !sourcePath.startsWith(prefix)
    || sourcePath.includes("\\")
    || sourcePath.split("/").some((part) => !part || part === "." || part === "..")
  ) {
    throw new Error("filePath must name one exact file under phase_0/");
  }
  return `phase_1/${sourcePath.slice(prefix.length)}`;
}
