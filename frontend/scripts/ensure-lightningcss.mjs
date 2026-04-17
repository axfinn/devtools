import fs from 'node:fs'
import path from 'node:path'
import { execSync } from 'node:child_process'
import { familySync, MUSL } from 'detect-libc'

const root = path.resolve(import.meta.dirname, '..')

function platformParts() {
  const parts = [process.platform, process.arch]

  if (process.platform === 'linux') {
    const family = familySync()
    if (family === MUSL) {
      parts.push('musl')
    } else if (process.arch === 'arm') {
      parts.push('gnueabihf')
    } else {
      parts.push('gnu')
    }
  } else if (process.platform === 'win32') {
    parts.push('msvc')
  }

  return parts
}

function ensureOptionalNativePackage(packageName, version) {
  const packageDir = path.join(root, 'node_modules', ...packageName.split('/'))

  if (fs.existsSync(packageDir)) {
    return
  }

  console.log(`[postinstall] installing missing ${packageName}@${version}`)
  execSync(`npm install --no-save --ignore-scripts ${packageName}@${version}`, {
    cwd: root,
    stdio: 'inherit',
  })
}

const parts = platformParts()

const targets = [
  {
    packageJson: path.join(root, 'node_modules', 'lightningcss', 'package.json'),
    packageName: `lightningcss-${parts.join('-')}`,
  },
  {
    packageJson: path.join(root, 'node_modules', '@tailwindcss', 'oxide', 'package.json'),
    packageName: `@tailwindcss/oxide-${parts.join('-')}`,
  },
]

for (const target of targets) {
  if (!fs.existsSync(target.packageJson)) {
    continue
  }
  const pkg = JSON.parse(fs.readFileSync(target.packageJson, 'utf8'))
  ensureOptionalNativePackage(target.packageName, pkg.version)
}
