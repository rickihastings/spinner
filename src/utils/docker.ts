import { execSync } from 'child_process';
import { writeFileSync, mkdirSync, copyFileSync, existsSync } from 'fs';
import { join, dirname } from 'path';
import { tmpdir, homedir } from 'os';
import { fileURLToPath } from 'url';
import { generateDockerfile } from './dockerfile.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface SpinConfig {
  image: string;
  repo: string;
  prompt?: string;
  branch?: string;
  maxIterations?: string;
}

export interface ValidationResult {
  valid: boolean;
  error?: string;
  warnings: string[];
  hasNpmrc: boolean;
}

export interface ContainerResult {
  success: boolean;
  containerName: string;
  error?: string;
}

export interface BuildConfig {
  name: string;
  baseImage?: string;
  dockerfile?: string;
}

/**
 * Escapes a string for safe use as a shell argument.
 * Wraps the string in single quotes and escapes any single quotes within.
 */
function escapeShellArg(arg: string): string {
  return `'${arg.replace(/'/g, "'\\''")}'`;
}

/**
 * Converts SSH Git URLs to HTTPS format for GitHub PAT authentication.
 * Example: git@github.com:user/repo.git -> https://github.com/user/repo.git
 */
function convertSshToHttps(repoUrl: string): string {
  if (repoUrl.startsWith('git@github.com:')) {
    return repoUrl.replace(/^git@github\.com:/, 'https://github.com/');
  }
  return repoUrl;
}

export function buildImage(config: BuildConfig): void {
  const buildContext = join(tmpdir(), `spinner-${Date.now()}`);
  mkdirSync(buildContext, { recursive: true });

  // Determine the base image to use
  let baseImage = config.baseImage || 'ubuntu:22.04';

  // If user provided a Dockerfile, build it first
  if (config.dockerfile) {
    const userBaseImageTag = `spinner-base:${config.name}`;
    execSync(`docker build -t ${userBaseImageTag} -f ${config.dockerfile} .`, {
      stdio: 'inherit',
    });
    baseImage = userBaseImageTag;
  }

  // Generate the final Dockerfile
  const dockerfilePath = join(buildContext, 'Dockerfile');
  const dockerfile = generateDockerfile({ baseImage });
  writeFileSync(dockerfilePath, dockerfile);

  // Copy skill templates and startup script to build context
  const templatesDir = join(buildContext, 'templates');
  const skillsDir = join(templatesDir, 'skills', 'task-implementation-lifecycle');
  const scriptsDir = join(templatesDir, 'scripts');
  mkdirSync(skillsDir, { recursive: true });
  mkdirSync(scriptsDir, { recursive: true });

  const skillTemplateSrc = join(
    __dirname,
    '../../templates/skills/task-implementation-lifecycle/SKILL.md',
  );
  const skillTemplateDest = join(skillsDir, 'SKILL.md');
  copyFileSync(skillTemplateSrc, skillTemplateDest);

  const startupScriptSrc = join(__dirname, '../../templates/scripts/startup.sh');
  const startupScriptDest = join(scriptsDir, 'startup.sh');
  copyFileSync(startupScriptSrc, startupScriptDest);

  const ralphLoopScriptSrc = join(__dirname, '../../templates/scripts/ralph-loop.sh');
  const ralphLoopScriptDest = join(scriptsDir, 'ralph-loop.sh');
  copyFileSync(ralphLoopScriptSrc, ralphLoopScriptDest);

  const imageName = `spinner:${config.name}`;
  execSync(`docker build -t ${imageName} .`, {
    cwd: buildContext,
    stdio: 'inherit',
  });
}

/**
 * Validates prerequisites for spinning up a container.
 * Checks: Docker image exists, valid git repo URL, GITHUB_TOKEN is set, CLAUDE_CODE_OAUTH_TOKEN is set, npmrc availability.
 */
export function validatePrerequisites(config: SpinConfig): ValidationResult {
  const warnings: string[] = [];

  // Check if repo is a valid git URL
  const isValidGitUrl =
    config.repo.startsWith('http://') ||
    config.repo.startsWith('https://') ||
    config.repo.startsWith('git@');
  if (!isValidGitUrl) {
    return {
      valid: false,
      error: 'Repository must be a valid git URL (https://, http://, or git@)',
      warnings,
      hasNpmrc: false,
    };
  }

  // Check if Docker image exists
  try {
    execSync(`docker image inspect ${config.image}`, { stdio: 'ignore' });
  } catch {
    return {
      valid: false,
      error: `Docker image '${config.image}' not found`,
      warnings,
      hasNpmrc: false,
    };
  }

  // Check GITHUB_TOKEN
  const githubToken = process.env.GITHUB_TOKEN;
  if (!githubToken) {
    return {
      valid: false,
      error:
        'GITHUB_TOKEN environment variable not set. Please set GITHUB_TOKEN before running spin.',
      warnings,
      hasNpmrc: false,
    };
  }

  // Check CLAUDE_CODE_OAUTH_TOKEN
  const claudeToken = process.env.CLAUDE_CODE_OAUTH_TOKEN;
  if (!claudeToken) {
    return {
      valid: false,
      error:
        'CLAUDE_CODE_OAUTH_TOKEN environment variable not set. Please set CLAUDE_CODE_OAUTH_TOKEN before running spin.',
      warnings,
      hasNpmrc: false,
    };
  }

  // Check ~/.npmrc
  const npmrcPath = join(homedir(), '.npmrc');
  const hasNpmrc = existsSync(npmrcPath);
  if (!hasNpmrc) {
    warnings.push('~/.npmrc not found, npm will use default registry');
  }

  return {
    valid: true,
    warnings,
    hasNpmrc,
  };
}

/**
 * Generates a unique container name from the repository URL.
 */
export function generateContainerName(repo: string): string {
  const repoName =
    repo
      .split('/')
      .pop()
      ?.replace(/\.git$/, '') || 'sandbox';
  const timestamp = Date.now();
  return `${repoName}-${timestamp}`;
}

/**
 * Builds the docker run command arguments.
 */
export function buildDockerRunCommand(
  config: SpinConfig,
  containerName: string,
  hasNpmrc: boolean,
): string[] {
  // Convert SSH URLs to HTTPS for GitHub PAT authentication
  const repoUrl = convertSshToHttps(config.repo);

  const dockerArgs = [
    'run',
    '-d',
    '--name',
    containerName,
    '-e',
    `GITHUB_TOKEN=${process.env.GITHUB_TOKEN}`,
    '-e',
    `CLAUDE_CODE_OAUTH_TOKEN=${process.env.CLAUDE_CODE_OAUTH_TOKEN}`,
    '-e',
    `REPO_URL=${repoUrl}`,
  ];

  // Add Ralph loop environment variables if prompt is provided
  if (config.prompt) {
    dockerArgs.push('-e', `PROMPT=${escapeShellArg(config.prompt)}`);
    dockerArgs.push('-e', `MAX_ITERATIONS=${config.maxIterations || '100'}`);

    // Add branch if specified
    if (config.branch) {
      dockerArgs.push('-e', `BRANCH=${escapeShellArg(config.branch)}`);
    }
  }

  // Add .npmrc mount if it exists
  if (hasNpmrc) {
    const npmrcPath = join(homedir(), '.npmrc');
    dockerArgs.push('-v', `${npmrcPath}:/home/spinner/.npmrc`);
  }

  // Add image
  dockerArgs.push(config.image);

  return dockerArgs;
}

/**
 * Executes the docker run command.
 */
export function executeDockerRun(dockerArgs: string[], containerName: string): ContainerResult {
  try {
    execSync(`docker ${dockerArgs.join(' ')}`, {
      stdio: 'pipe',
      encoding: 'utf-8',
    });

    return {
      success: true,
      containerName,
    };
  } catch (error: unknown) {
    // Container may have started but clone failed
    // Try to get the git error message from container logs
    try {
      const logs = execSync(`docker logs ${containerName}`, {
        encoding: 'utf-8',
        stdio: 'pipe',
      });
      return {
        success: false,
        containerName,
        error: `Git clone failed: ${logs.trim()}`,
      };
    } catch {
      const message = error instanceof Error ? error.message : 'Failed to create container';
      const stderr =
        error && typeof error === 'object' && 'stderr' in error ? String(error.stderr) : '';
      return {
        success: false,
        containerName,
        error: stderr || message,
      };
    }
  }
}

/**
 * Verifies that the container is running.
 */
export function verifyContainerStatus(containerName: string): ContainerResult {
  try {
    const containerStatus = execSync(`docker inspect -f '{{.State.Status}}' ${containerName}`, {
      encoding: 'utf-8',
      stdio: 'pipe',
    }).trim();

    if (containerStatus !== 'running') {
      // Get logs to show what went wrong
      const logs = execSync(`docker logs ${containerName}`, {
        encoding: 'utf-8',
        stdio: 'pipe',
      });
      return {
        success: false,
        containerName,
        error: `Container exited. Logs: ${logs.trim()}`,
      };
    }

    return {
      success: true,
      containerName,
    };
  } catch {
    return {
      success: false,
      containerName,
      error: 'Failed to verify container status',
    };
  }
}
