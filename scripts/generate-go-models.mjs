#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { parse } from 'yaml';

const [, , sourcePath, outputPath] = process.argv;

if (!sourcePath || !outputPath) {
  console.error('usage: generate-go-models.mjs <openapi-source> <output>');
  process.exit(2);
}

const document = parse(fs.readFileSync(sourcePath, 'utf8'));
const schemas = document.components?.schemas;

if (!schemas || typeof schemas !== 'object') {
  throw new Error(`OpenAPI source has no component schemas: ${sourcePath}`);
}

const exportedName = (name) => name
  .replace(/(^|[-_ ])([a-z])/g, (_, __, letter) => letter.toUpperCase())
  .replace(/Id$/, 'ID')
  .replace(/Url$/, 'URL');

const primitiveType = (schema) => {
  if (schema.format === 'date-time') return 'time.Time';
  if (schema.type === 'string') return 'string';
  if (schema.type === 'integer') return 'int32';
  if (schema.type === 'number') return 'float64';
  if (schema.type === 'boolean') return 'bool';
  return 'any';
};

const isNullable = (schema) => schema?.oneOf?.some((item) => item.type === 'null') === true;

const goType = (schema) => {
  if (!schema) return 'any';
  if (schema.const !== undefined) return 'string';
  if (schema.$ref) return schema.$ref.split('/').at(-1);
  if (schema.oneOf) {
    const nonNull = schema.oneOf.filter((item) => item.type !== 'null');
    if (nonNull.length === 1 && schema.oneOf.length !== 1) return goType(nonNull[0]);
    return 'any';
  }
  if (schema.type === 'array') return `[]${goType(schema.items)}`;
  if (schema.type === 'object' && schema.additionalProperties) {
    const valueType = schema.additionalProperties.type === 'string' ? 'string' : 'any';
    return `map[string]${valueType}`;
  }
  if (schema.type === 'object') return 'map[string]any';
  return primitiveType(schema);
};

const propertyType = (schema, required) => {
  const type = goType(schema);
  if (isNullable(schema)) return `*${type}`;
  if (required) return type;
  return `*${type}`;
};

const comments = (description) => (description ? `// ${description.replaceAll('\n', ' ')}` : '');
const lines = [
  '// Code generated from internal/contracts/api.openapi.yaml; DO NOT EDIT.',
  '',
  'package contracts',
  '',
  'import "time"',
  '',
];
const unionMethods = [];

for (const [name, schema] of Object.entries(schemas).sort(([a], [b]) => a.localeCompare(b))) {
  const typeName = exportedName(name);
  if (schema.enum) {
    lines.push(`type ${typeName} string`, '');
    for (const value of schema.enum) {
      lines.push(`const ${typeName}${exportedName(String(value))} ${typeName} = ${JSON.stringify(value)}`);
    }
    lines.push('');
    continue;
  }

  if (schema.oneOf?.every((item) => item.$ref)) {
    lines.push(`type ${typeName} interface {`, `\tis${typeName}()`, '}', '');
    for (const item of schema.oneOf) unionMethods.push(`func (${item.$ref.split('/').at(-1)}) is${typeName}() {}`);
    continue;
  }

  if (schema.type !== 'object' || !schema.properties) {
    lines.push(`type ${typeName} = ${goType(schema)}`, '');
    continue;
  }

  const required = new Set(schema.required ?? []);
  lines.push(`type ${typeName} struct {`);
  for (const [property, propertySchema] of Object.entries(schema.properties)) {
    const fieldName = exportedName(property);
    const description = comments(propertySchema.description);
    if (description) lines.push(`\t${description}`);
    lines.push(`\t${fieldName} ${propertyType(propertySchema, required.has(property))} \`json:"${property}${required.has(property) ? '' : ',omitempty'}"\``);
  }
  lines.push('}', '');
}

lines.push(...unionMethods, '');

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, `${lines.join('\n')}\n`);
execFileSync('gofmt', ['-w', outputPath]);
