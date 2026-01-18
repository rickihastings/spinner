import { execSync } from 'child_process';
import { writeFileSync, mkdirSync, copyFileSync } from 'fs';
import { join, dirname } from 'path';
import { tmpdir } from 'os';
import { fileURLToPath } from 'url';
import { generateDockerfile, DockerfileConfig } from './dockerfile.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface BuildConfig extends DockerfileConfig {
  name: string;
}

export function buildImage(config: BuildConfig): void {
  const buildContext = join(tmpdir(), `spinner-${Date.now()}`);
  mkdirSync(buildContext, { recursive: true });

  const dockerfilePath = join(buildContext, 'Dockerfile');
  const dockerfile = generateDockerfile({
    nodeVersion: config.nodeVersion,
    jvmUrl: config.jvmUrl,
  });
  writeFileSync(dockerfilePath, dockerfile);

  // Copy skill templates and startup script to build context
  const templatesDir = join(buildContext, 'templates');
  mkdirSync(templatesDir, { recursive: true });

  const skillTemplateSrc = join(
    __dirname,
    '../../templates/task-implementation-lifecycle.skill.md',
  );
  const skillTemplateDest = join(templatesDir, 'task-implementation-lifecycle.skill.md');
  copyFileSync(skillTemplateSrc, skillTemplateDest);

  const startupScriptSrc = join(__dirname, '../../templates/startup.sh');
  const startupScriptDest = join(templatesDir, 'startup.sh');
  copyFileSync(startupScriptSrc, startupScriptDest);

  const imageName = `spinner:${config.name}`;
  execSync(`docker build --no-cache -t ${imageName} .`, {
    cwd: buildContext,
    stdio: 'inherit',
  });
}
