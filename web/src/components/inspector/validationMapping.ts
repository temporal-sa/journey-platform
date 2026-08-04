import type { ValidationResult, ValidationIssue } from '../../types/api';

/**
 * Filter validation issues for a specific node ID.
 */
export function getNodeIssues(
  nodeId: string | null | undefined,
  validationResult: ValidationResult | null | undefined
): ValidationIssue[] {
  if (!nodeId || !validationResult?.issues) return [];
  return validationResult.issues.filter((issue) => issue.node_id === nodeId);
}

/**
 * Get field-level validation issue for a specific field path in a node.
 * Matches paths like 'config.recipient', 'recipient', 'name', etc.
 */
export function getFieldError(
  fieldPath: string,
  nodeIssues: ValidationIssue[]
): ValidationIssue | undefined {
  if (!fieldPath || !nodeIssues.length) return undefined;

  const normalizedTarget = fieldPath.replace(/^config\./, '').toLowerCase();

  return nodeIssues.find((issue) => {
    const normIssuePath = issue.field_path.replace(/^config\./, '').toLowerCase();
    return normIssuePath === normalizedTarget || issue.field_path.toLowerCase() === fieldPath.toLowerCase();
  });
}

/**
 * Calculate error and warning count for a specific canvas node.
 */
export function getNodeValidationCounts(
  nodeId: string | null | undefined,
  validationResult: ValidationResult | null | undefined
): { errors: number; warnings: number } {
  const nodeIssues = getNodeIssues(nodeId, validationResult);
  let errors = 0;
  let warnings = 0;

  for (const issue of nodeIssues) {
    if (issue.severity === 'error') errors++;
    else if (issue.severity === 'warning') warnings++;
  }

  return { errors, warnings };
}

/**
 * Calculate total error and warning count for all nodes in the validation result.
 */
export function getTabErrorCounts(
  validationResult: ValidationResult | null | undefined
): { errors: number; warnings: number } {
  if (!validationResult?.issues) return { errors: 0, warnings: 0 };

  let errors = 0;
  let warnings = 0;

  for (const issue of validationResult.issues) {
    if (issue.severity === 'error') errors++;
    else if (issue.severity === 'warning') warnings++;
  }

  return { errors, warnings };
}

/**
 * Format machine error code into readable tag (e.g. ERR_MISSING_FIELD -> Missing Field).
 */
export function formatIssueCode(code: string): string {
  if (!code) return 'Validation Error';
  return code
    .replace(/^ERR_|^WARN_/, '')
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ');
}
