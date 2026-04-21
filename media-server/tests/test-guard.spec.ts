import { describe, expect, it } from 'vitest';

import { assertTestOnly } from '../src/utils/test-guard.js';

describe('assertTestOnly', () => {
  it('passes when NODE_ENV=test', () => {
    expect(() => assertTestOnly('sampleFn')).not.toThrow();
  });

  it('throws when NODE_ENV is not test', () => {
    const original = process.env.NODE_ENV;
    process.env.NODE_ENV = 'production';
    try {
      expect(() => assertTestOnly('sampleFn')).toThrow(/test-only/);
    } finally {
      process.env.NODE_ENV = original;
    }
  });
});
