import { describe, it, expect } from "vitest";
import {
  isProgressShape,
  isTodayShape,
  isStatsShape,
} from "@/types/dashboard";

describe("dashboard type guards", () => {
  describe("isProgressShape", () => {
    it("returns true for valid {index, total}", () => {
      expect(isProgressShape({ index: 3, total: 10 })).toBe(true);
    });

    it("returns false for invalid data", () => {
      expect(isProgressShape({ foo: 1 })).toBe(false);
      expect(isProgressShape(null)).toBe(false);
      expect(isProgressShape(undefined)).toBe(false);
      expect(isProgressShape("string")).toBe(false);
      expect(isProgressShape({ index: "3", total: 10 })).toBe(false);
    });
  });

  describe("isTodayShape", () => {
    it("returns true for valid {count, target}", () => {
      expect(isTodayShape({ count: 5, target: 20 })).toBe(true);
    });
  });

  describe("isStatsShape", () => {
    it("returns true for valid {followed, already, skipped, error}", () => {
      expect(
        isStatsShape({ followed: 10, already: 3, skipped: 2, error: 1 })
      ).toBe(true);
    });
  });

  describe("guard failure fallback", () => {
    it("JSON.stringify works as fallback for unrecognized shapes", () => {
      const unknownData = { custom: "data", nested: { key: "value" } };

      // When guards fail, the caller uses JSON.stringify
      expect(isProgressShape(unknownData)).toBe(false);
      expect(isTodayShape(unknownData)).toBe(false);
      expect(isStatsShape(unknownData)).toBe(false);

      // Fallback: stringify should produce valid output
      const fallback = JSON.stringify(unknownData);
      expect(fallback).toBe('{"custom":"data","nested":{"key":"value"}}');
    });
  });
});
