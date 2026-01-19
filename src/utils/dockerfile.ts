import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface DockerfileConfig {
  baseImage: string;
}

export function generateDockerfile(config: DockerfileConfig): string {
  const templatePath = join(__dirname, '../../templates/docker/extending.template');
  const template = readFileSync(templatePath, 'utf-8');

  return template.replace(/\$\{BASE_IMAGE\}/g, config.baseImage);
}
