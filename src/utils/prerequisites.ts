import { execSync } from 'child_process';

export interface Prerequisite {
  name: string;
  command: string;
  errorMessage: string;
}

const PREREQUISITES: Prerequisite[] = [
  {
    name: 'Docker',
    command: 'docker --version',
    errorMessage: 'Docker is not installed. Please install Docker Desktop.',
  },
  {
    name: 'Git',
    command: 'git --version',
    errorMessage: 'Git is not installed. Please install Git.',
  },
  {
    name: 'Claude',
    command: 'claude --version',
    errorMessage: 'Claude CLI is not installed. Please install claude-code.',
  },
];

export class PrerequisiteError extends Error {
  constructor(public prerequisite: Prerequisite) {
    super(prerequisite.errorMessage);
    this.name = 'PrerequisiteError';
  }
}

export function checkPrerequisites(): void {
  for (const prereq of PREREQUISITES) {
    try {
      execSync(prereq.command, { stdio: 'ignore' });
    } catch {
      throw new PrerequisiteError(prereq);
    }
  }
}
