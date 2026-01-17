import { execSync } from 'child_process';
import { writeFileSync, mkdirSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { generateDockerfile, DockerfileConfig } from './dockerfile.js';

export interface BuildConfig extends DockerfileConfig {
  name: string;
}

export function buildImage(config: BuildConfig): void {
  const buildContext = join(tmpdir(), `docker-sandbox-${Date.now()}`);
  mkdirSync(buildContext, { recursive: true });

  const dockerfilePath = join(buildContext, 'Dockerfile');
  const dockerfile = generateDockerfile({
    nodeVersion: config.nodeVersion,
    jvmUrl: config.jvmUrl,
  });
  writeFileSync(dockerfilePath, dockerfile);

  const imageName = `docker-sandbox:${config.name}`;
  execSync(`docker build --no-cache -t ${imageName} .`, {
    cwd: buildContext,
    stdio: 'inherit',
  });
}
