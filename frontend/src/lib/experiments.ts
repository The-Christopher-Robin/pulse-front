export type Variant = {
  key: string;
  name: string;
  weight: number;
};

export type Assignment = {
  experiment_key: string;
  variant_key: string;
  user_id: string;
  occurred_at: string;
  exposed: boolean;
};

export type AssignmentMap = Record<string, Assignment>;

export const HOLDOUT = 'holdout';

export function variantFor(assignments: AssignmentMap, experimentKey: string): string {
  const a = assignments[experimentKey];
  if (!a) return HOLDOUT;
  return a.variant_key;
}

export function inTreatment(assignments: AssignmentMap, experimentKey: string, ...treatmentKeys: string[]): boolean {
  const v = variantFor(assignments, experimentKey);
  if (!treatmentKeys.length) return v !== HOLDOUT && v !== 'control';
  return treatmentKeys.includes(v);
}
