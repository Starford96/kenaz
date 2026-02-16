#!/usr/bin/env node
// Converts Swagger 2.0 JSON to OpenAPI 3.1 YAML.
// Run from repo root: node scripts/swagger2openapi.mjs <in.json> <out.yaml>
import { readFileSync, writeFileSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { createRequire } from "module";

// Resolve packages from frontend/node_modules.
const __dirname = dirname(fileURLToPath(import.meta.url));
const req = createRequire(resolve(__dirname, "../frontend/package.json"));
const converter = req("swagger2openapi");
const yaml = req("yaml");

const input = JSON.parse(readFileSync(process.argv[2], "utf8"));

const { openapi } = await converter.convertObj(input, {
  patch: true,
  warnOnly: true,
});

// Override to 3.1.0
openapi.openapi = "3.1.0";

// Add server block
openapi.servers = [{ url: "/api", description: "Default (relative)" }];

// Strip Go package prefix ("api.") from schema names for cleaner types.
if (openapi.components?.schemas) {
  const schemas = openapi.components.schemas;
  const renames = {};
  for (const key of Object.keys(schemas)) {
    const clean = key.replace(/^api\./, "");
    if (clean !== key) renames[key] = clean;
  }
  for (const [old, nw] of Object.entries(renames)) {
    schemas[nw] = schemas[old];
    delete schemas[old];
  }
  // Fix all $ref pointers.
  const raw = JSON.stringify(openapi);
  const fixed = raw.replace(/#\/components\/schemas\/api\./g, "#/components/schemas/");
  Object.assign(openapi, JSON.parse(fixed));
}

writeFileSync(process.argv[3], yaml.stringify(openapi, { lineWidth: 120 }));
console.log(`Wrote ${process.argv[3]}`);
