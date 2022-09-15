import fs from 'node:fs/promises'
import { URL } from 'node:url'
import pkg from '../package.json'
// @ts-ignore
import greaterThan from 'semver/functions/gt'

async function run(version: string) {
  const url = new URL('../package.json', import.meta.url)

  if (!greaterThan(version, pkg.version)) {
    console.error(`Version ${version} is not greater than ${pkg.version}.`)
    process.exit(1)
  }

  pkg.version = version

  await fs.writeFile(url, JSON.stringify(pkg, null, 2))

  console.info(`Updated ${pkg.name} to ${version}.`)
}

run(process.argv[process.argv.length - 1])
