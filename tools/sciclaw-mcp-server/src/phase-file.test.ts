import { expect, test } from "bun:test";
import { phaseOnePath } from "./phase-file.js";

test("maps only exact phase_0 file paths", () => {
  expect(phaseOnePath("phase_0/procedures/sop.md")).toBe("phase_1/procedures/sop.md");
  expect(() => phaseOnePath("phase_0/../secret.md")).toThrow();
});
