import { expect, test } from "bun:test";
import { allPages } from "./pagination.js";

test("collects every nonempty page", async () => {
  const pages = [["one"], ["two", "three"], []];
  expect(await allPages(async (page) => pages[page - 1] ?? [])).toEqual(["one", "two", "three"]);
});
