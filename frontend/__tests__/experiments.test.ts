import { variantFor, inTreatment, HOLDOUT, type AssignmentMap } from '../src/lib/experiments';

function makeAssignments(): AssignmentMap {
  return {
    landing_hero_copy: {
      experiment_key: 'landing_hero_copy',
      variant_key: 'treatment',
      user_id: 'u1',
      occurred_at: new Date().toISOString(),
      exposed: true,
    },
    product_card_badge: {
      experiment_key: 'product_card_badge',
      variant_key: 'control',
      user_id: 'u1',
      occurred_at: new Date().toISOString(),
      exposed: true,
    },
  };
}

describe('variantFor', () => {
  it('returns the assigned variant when present', () => {
    expect(variantFor(makeAssignments(), 'landing_hero_copy')).toBe('treatment');
  });

  it('returns holdout when the experiment is missing', () => {
    expect(variantFor(makeAssignments(), 'missing_key')).toBe(HOLDOUT);
  });
});

describe('inTreatment', () => {
  it('recognises any non-control, non-holdout variant with no whitelist', () => {
    expect(inTreatment(makeAssignments(), 'landing_hero_copy')).toBe(true);
    expect(inTreatment(makeAssignments(), 'product_card_badge')).toBe(false);
  });

  it('honours an explicit whitelist of variant keys', () => {
    expect(inTreatment(makeAssignments(), 'product_card_badge', 'control')).toBe(true);
    expect(inTreatment(makeAssignments(), 'landing_hero_copy', 'treatment_a')).toBe(false);
  });
});
