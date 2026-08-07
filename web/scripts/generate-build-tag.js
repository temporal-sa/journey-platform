import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export function getGitInfo() {
  try {
    const toplevel = execSync('git rev-parse --show-toplevel', { encoding: 'utf-8' }).trim();
    const worktree = path.basename(toplevel);
    const branch = execSync('git rev-parse --abbrev-ref HEAD', { encoding: 'utf-8' }).trim();
    return { worktree, branch };
  } catch (err) {
    return { worktree: 'unknown-worktree', branch: 'unknown-branch' };
  }
}

export function updateBuildIncrement(branch) {
  // Keep track of change increment per branch on disk in a text file
  const incrementsFilePath = path.resolve(__dirname, '../../.agents/build_increments.txt');

  let branchIncrements = {};
  if (fs.existsSync(incrementsFilePath)) {
    try {
      const content = fs.readFileSync(incrementsFilePath, 'utf-8');
      content.split('\n').forEach(line => {
        const [b, inc] = line.split('=');
        if (b && inc) {
          branchIncrements[b.trim()] = parseInt(inc.trim(), 10) || 0;
        }
      });
    } catch (e) {
      branchIncrements = {};
    }
  }

  const currentInc = (branchIncrements[branch] || 0) + 1;
  branchIncrements[branch] = currentInc;

  // Save updated increments to text file on disk
  const newContent = Object.entries(branchIncrements)
    .map(([b, inc]) => `${b}=${inc}`)
    .join('\n');

  fs.writeFileSync(incrementsFilePath, newContent + '\n', 'utf-8');
  return currentInc;
}

export function generateBuildTag() {
  const { worktree, branch } = getGitInfo();
  const increment = updateBuildIncrement(branch);
  const buildTag = `${worktree}-${branch}-${increment}`;

  const outputPath = path.resolve(__dirname, '../src/buildTag.ts');
  const code = `// Auto-generated build tag file
export const BUILD_TAG = '${buildTag}';
export const WORKTREE_NAME = '${worktree}';
export const BRANCH_NAME = '${branch}';
export const BUILD_INCREMENT = ${increment};
`;

  fs.writeFileSync(outputPath, code, 'utf-8');
  console.log(`✅ Build tag generated: ${buildTag}`);
  return buildTag;
}

// Run directly if invoked via CLI
if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(__filename)) {
  generateBuildTag();
}
