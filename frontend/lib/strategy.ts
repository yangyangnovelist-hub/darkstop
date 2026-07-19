export function percentToBps(value: string): number {
  if (!/^\d{1,2}(?:\.\d{1,2})?$|^100(?:\.0{1,2})?$/.test(value)) {
    throw new Error("Trailing distance accepts at most two decimal places.");
  }
  const [whole, fraction = ""] = value.split(".");
  const bps = Number(whole) * 100 + Number(fraction.padEnd(2, "0"));
  if (bps < 25 || bps > 5000) {
    throw new Error("Trailing distance must be between 0.25% and 50%.");
  }
  return bps;
}
