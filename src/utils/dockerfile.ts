import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface DockerfileConfig {
  nodeVersion: string;
  jvmUrl: string;
}

export function generateDockerfile(config: DockerfileConfig): string {
  const templatePath = join(__dirname, '../../templates/Dockerfile.template');
  const template = readFileSync(templatePath, 'utf-8');

  return template
    .replace(/{{NODE_VERSION}}/g, config.nodeVersion)
    .replace(/{{JVM_URL}}/g, config.jvmUrl);
}
