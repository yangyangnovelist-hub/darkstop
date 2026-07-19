import { describe, expect, it } from "vitest";
import { percentToBps } from "./strategy";

describe("percentToBps", () => {
  it.each([
    ["0.25", 25],
    ["5", 500],
    ["5.5", 550],
    ["50.00", 5000],
  ])("parses %s exactly", (value, expected) => {
    expect(percentToBps(value)).toBe(expected);
  });

  it.each(["0", "0.249", "5.001", "50.01", "100", "abc", ""])(
    "rejects %s without rounding",
    (value) => {
      expect(() => percentToBps(value)).toThrow();
    },
  );
});
