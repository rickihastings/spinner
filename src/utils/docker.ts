import { execSync } from 'child_process';
import { writeFileSync, mkdirSync, copyFileSync } from 'fs';
import { join, dirname } from 'path';
import { tmpdir } from 'os';
import { fileURLToPath } from 'url';
import { generateDockerfile } from './dockerfile.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface BuildConfig {
  name: string;
  baseImage?: string;
  dockerfile?: string;
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
  const skillsDir = join(templatesDir, 'skills');
  const scriptsDir = join(templatesDir, 'scripts');
  mkdirSync(skillsDir, { recursive: true });
  mkdirSync(scriptsDir, { recursive: true });

  const skillTemplateSrc = join(
    __dirname,
    '../../templates/skills/task-implementation-lifecycle.skill.md',
  );
  const skillTemplateDest = join(skillsDir, 'task-implementation-lifecycle.skill.md');
  copyFileSync(skillTemplateSrc, skillTemplateDest);

  const startupScriptSrc = join(__dirname, '../../templates/scripts/startup.sh');
  const startupScriptDest = join(scriptsDir, 'startup.sh');
  copyFileSync(startupScriptSrc, startupScriptDest);

  const imageName = `spinner:${config.name}`;
  execSync(`docker build --no-cache -t ${imageName} .`, {
    cwd: buildContext,
    stdio: 'inherit',
  });
}
