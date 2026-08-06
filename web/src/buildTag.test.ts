import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';
import { BUILD_TAG, WORKTREE_NAME, BRANCH_NAME, BUILD_INCREMENT } from './buildTag';

describe('Build Tag Generator & Increments System', () => {
  it('exports valid build tag constants', () => {
    expect(BUILD_TAG).toBeDefined();
    expect(WORKTREE_NAME).toBeDefined();
    expect(BRANCH_NAME).toBeDefined();
    expect(BUILD_INCREMENT).toBeGreaterThan(0);

    const expectedFormat = `${WORKTREE_NAME}-${BRANCH_NAME}-${BUILD_INCREMENT}`;
    expect(BUILD_TAG).toBe(expectedFormat);
  });

  it('persists change increment per branch on disk in text file', () => {
    const incrementsFilePath = path.resolve(__dirname, '../../.agents/build_increments.txt');
    expect(fs.existsSync(incrementsFilePath)).toBe(true);

    const content = fs.readFileSync(incrementsFilePath, 'utf-8');
    expect(content).toContain(`${BRANCH_NAME}=`);
  });
});
