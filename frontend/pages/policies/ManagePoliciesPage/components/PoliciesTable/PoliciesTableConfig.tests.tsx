import { nextPolicyUpdateMs } from "./PoliciesTableConfig";

describe("Next policy update", () => {
  it("when all zero", () => {
    expect(nextPolicyUpdateMs(new Date(0), 0, 0, 0)).toBe(0);
  });
  it("when all zero except next update", () => {
    expect(nextPolicyUpdateMs(new Date(0), 10, 0, 0)).toBe(10);
  });
  it("on next node count update", () => {
    expect(nextPolicyUpdateMs(new Date(), 10, 200, 5)).toBe(10);
  });
  it("on next node count update with recent policy", () => {
    expect(
      nextPolicyUpdateMs(new Date(new Date().getTime() - 91), 10, 200, 100)
    ).toBe(10);
  });
  it("on next node count update with old policy", () => {
    expect(nextPolicyUpdateMs(new Date("2020-01-01"), 10, 200, 100)).toBe(10);
  });
  it("on subsequent node count update", () => {
    expect(nextPolicyUpdateMs(new Date(), 10, 200, 20)).toBe(210);
  });
  it("on far future node count update", () => {
    expect(nextPolicyUpdateMs(new Date(), 10, 200, 1009)).toBe(1010);
  });
});
