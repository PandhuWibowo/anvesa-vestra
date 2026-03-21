/**
 * Comprehensive file type detection and categorization.
 * Each function checks both Content-Type header and file extension.
 */

function ext(entry) {
  const name = entry?.display || entry?.name || ''
  const dot = name.lastIndexOf('.')
  return dot >= 0 ? name.slice(dot + 1).toLowerCase() : ''
}

function ct(entry) {
  return (entry?.content_type || '').toLowerCase()
}

// ── Images ────────────────────────────────────────────────────────

const IMAGE_EXTS = new Set([
  'jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'ico', 'bmp',
  'tiff', 'tif', 'avif', 'heic', 'heif', 'jfif', 'apng',
])

export function isImage(entry) {
  return ct(entry).startsWith('image/') || IMAGE_EXTS.has(ext(entry))
}

// ── Video ─────────────────────────────────────────────────────────

const VIDEO_EXTS = new Set([
  'mp4', 'webm', 'mov', 'avi', 'mkv', 'ogv', 'm4v', 'flv',
  '3gp', '3gpp', 'wmv', 'mpg', 'mpeg', 'ts', 'mts', 'm2ts',
])

export function isVideo(entry) {
  return ct(entry).startsWith('video/') || VIDEO_EXTS.has(ext(entry))
}

// ── Audio ─────────────────────────────────────────────────────────

const AUDIO_EXTS = new Set([
  'mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a', 'opus', 'wma',
  'mid', 'midi', 'aiff', 'aif', 'ape', 'alac', 'weba', 'amr',
])

export function isAudio(entry) {
  return ct(entry).startsWith('audio/') || AUDIO_EXTS.has(ext(entry))
}

// ── PDF ───────────────────────────────────────────────────────────

export function isPdf(entry) {
  return ct(entry) === 'application/pdf' || ext(entry) === 'pdf'
}

// ── Markdown ──────────────────────────────────────────────────────

const MD_EXTS = new Set(['md', 'markdown', 'mdx', 'mdown', 'mkd'])

export function isMarkdown(entry) {
  return MD_EXTS.has(ext(entry))
}

// ── JSON ──────────────────────────────────────────────────────────

const JSON_EXTS = new Set(['json', 'jsonl', 'ndjson', 'jsonc', 'json5', 'geojson', 'har'])

export function isJson(entry) {
  const c = ct(entry)
  return c.includes('json') || JSON_EXTS.has(ext(entry))
}

// ── CSV / Tabular ─────────────────────────────────────────────────

const CSV_EXTS = new Set(['csv', 'tsv'])

export function isCsv(entry) {
  return ct(entry) === 'text/csv' || CSV_EXTS.has(ext(entry))
}

// ── SVG (previewable as both image and code) ──────────────────────

export function isSvg(entry) {
  return ct(entry).includes('svg') || ext(entry) === 'svg'
}

// ── Code / Programming ────────────────────────────────────────────

const CODE_EXTS = new Set([
  // Web
  'js', 'mjs', 'cjs', 'jsx', 'ts', 'tsx', 'vue', 'svelte', 'astro',
  'html', 'htm', 'xhtml', 'css', 'scss', 'sass', 'less', 'styl', 'stylus',

  // Systems
  'c', 'h', 'cpp', 'cc', 'cxx', 'hpp', 'hxx', 'cs', 'go', 'rs', 'zig',
  'swift', 'kt', 'kts', 'java', 'scala', 'groovy', 'gradle',

  // Scripting
  'py', 'pyw', 'rb', 'php', 'pl', 'pm', 'lua', 'r', 'R', 'jl',
  'sh', 'bash', 'zsh', 'fish', 'bat', 'cmd', 'ps1', 'psm1',

  // Functional
  'ex', 'exs', 'clj', 'cljs', 'edn', 'elm', 'hs', 'lhs', 'ml', 'mli',
  'fs', 'fsx', 'erl', 'hrl',

  // Data / Query
  'sql', 'graphql', 'gql', 'proto', 'thrift', 'avsc',

  // Mobile
  'dart', 'kt', 'swift', 'm', 'mm',

  // Markup / Templates
  'jsx', 'tsx', 'hbs', 'ejs', 'pug', 'jade', 'mustache', 'njk',
  'jinja', 'jinja2', 'j2', 'twig',

  // Build / Infra
  'dockerfile', 'makefile', 'cmake', 'sbt', 'tf', 'hcl',
  'vagrantfile', 'rakefile', 'gemfile',

  // Config as code
  'nix', 'dhall', 'jsonnet', 'cue', 'rego',
])

export function isCode(entry) {
  const e = ext(entry)
  if (CODE_EXTS.has(e)) return true
  const c = ct(entry)
  return c.includes('javascript') || c.includes('typescript') ||
         c.includes('x-python') || c.includes('x-ruby') ||
         c.includes('x-shellscript') || c.includes('x-c') ||
         c.includes('x-java')
}

// ── Config / Data files ───────────────────────────────────────────

const CONFIG_EXTS = new Set([
  'yaml', 'yml', 'toml', 'ini', 'conf', 'cfg', 'env', 'properties',
  'editorconfig', 'prettierrc', 'eslintrc', 'babelrc', 'npmrc',
  'gitignore', 'gitattributes', 'dockerignore', 'hgignore',
  'browserslistrc', 'nvmrc', 'tool-versions',
])

export function isConfig(entry) {
  return CONFIG_EXTS.has(ext(entry))
}

// ── Plain text (general) ──────────────────────────────────────────

const TEXT_EXTS = new Set([
  'txt', 'log', 'out', 'diff', 'patch', 'rst', 'adoc', 'asciidoc',
  'tex', 'latex', 'org', 'nfo', 'changelog', 'license', 'readme',
  'authors', 'contributors', 'todo', 'copying', 'notice',
  'xml', 'xsl', 'xslt', 'xsd', 'dtd', 'rss', 'atom', 'opml',
  'plist', 'manifest', 'webmanifest',
])

export function isPlainText(entry) {
  const c = ct(entry)
  return c.startsWith('text/') || c.includes('xml') || TEXT_EXTS.has(ext(entry))
}

// ── Composite: "is this previewable as text?" ─────────────────────

export function isTextPreviewable(entry) {
  return isCode(entry) || isConfig(entry) || isPlainText(entry) ||
         isJson(entry) || isCsv(entry) || isMarkdown(entry)
}

// ── Font ──────────────────────────────────────────────────────────

const FONT_EXTS = new Set(['woff', 'woff2', 'ttf', 'otf', 'eot'])

export function isFont(entry) {
  return ct(entry).includes('font') || FONT_EXTS.has(ext(entry))
}

// ── Archive / Compressed ──────────────────────────────────────────

const ARCHIVE_EXTS = new Set([
  'zip', 'tar', 'gz', 'tgz', 'bz2', 'xz', 'rar', '7z', 'lz', 'zst',
  'lzma', 'cab', 'iso', 'dmg', 'deb', 'rpm', 'apk', 'jar', 'war', 'ear',
])

export function isArchive(entry) {
  const c = ct(entry)
  return c.includes('zip') || c.includes('compressed') || c.includes('archive') ||
         c.includes('tar') || c.includes('gzip') || ARCHIVE_EXTS.has(ext(entry))
}

// ── Excel / Spreadsheet ───────────────────────────────────────────

const EXCEL_EXTS = new Set(['xls', 'xlsx', 'xlsm', 'xlsb', 'ods', 'numbers'])

export function isExcel(entry) {
  const c = ct(entry)
  return c.includes('spreadsheet') || c.includes('ms-excel') ||
         c.includes('officedocument.spreadsheet') || EXCEL_EXTS.has(ext(entry))
}

// ── Word / Document ───────────────────────────────────────────────

const WORD_EXTS = new Set(['doc', 'docx', 'odt', 'rtf', 'pages'])

export function isWord(entry) {
  const c = ct(entry)
  return c.includes('msword') || c.includes('officedocument.wordprocessing') ||
         c.includes('opendocument.text') || WORD_EXTS.has(ext(entry))
}

// ── PowerPoint / Presentation ─────────────────────────────────────

const PPT_EXTS = new Set(['ppt', 'pptx', 'odp', 'keynote'])

export function isPowerPoint(entry) {
  const c = ct(entry)
  return c.includes('ms-powerpoint') || c.includes('officedocument.presentation') ||
         c.includes('opendocument.presentation') || PPT_EXTS.has(ext(entry))
}

// ── Office / Document (any of the above) ──────────────────────────

const OFFICE_EXTS = new Set([
  ...EXCEL_EXTS, ...WORD_EXTS, ...PPT_EXTS, 'epub',
])

export function isOffice(entry) {
  return isExcel(entry) || isWord(entry) || isPowerPoint(entry) || OFFICE_EXTS.has(ext(entry))
}

// ── Executable / Binary ───────────────────────────────────────────

const BINARY_EXTS = new Set([
  'exe', 'dll', 'so', 'dylib', 'bin', 'dat', 'o', 'a', 'lib',
  'wasm', 'class', 'pyc', 'pyd',
])

export function isBinary(entry) {
  return ct(entry) === 'application/octet-stream' && BINARY_EXTS.has(ext(entry))
}

// ── File category (for icon selection) ────────────────────────────

export function fileCategory(entry) {
  if (!entry || entry.type === 'dir') return 'folder'
  if (isImage(entry))      return 'image'
  if (isVideo(entry))      return 'video'
  if (isAudio(entry))      return 'audio'
  if (isPdf(entry))        return 'pdf'
  if (isExcel(entry))      return 'spreadsheet'
  if (isWord(entry))       return 'document'
  if (isPowerPoint(entry)) return 'presentation'
  if (isArchive(entry))    return 'archive'
  if (isFont(entry))       return 'font'
  if (isCode(entry))       return 'code'
  if (isJson(entry))       return 'data'
  if (isCsv(entry))        return 'spreadsheet'
  if (isMarkdown(entry))   return 'markdown'
  if (isConfig(entry))     return 'config'
  if (isPlainText(entry))  return 'text'
  if (isBinary(entry))     return 'binary'
  return 'file'
}

// ── Language label for syntax display ─────────────────────────────

const LANG_MAP = {
  js: 'JavaScript', mjs: 'JavaScript', cjs: 'JavaScript', jsx: 'JSX',
  ts: 'TypeScript', tsx: 'TSX', vue: 'Vue', svelte: 'Svelte',
  py: 'Python', pyw: 'Python', rb: 'Ruby', php: 'PHP',
  go: 'Go', rs: 'Rust', java: 'Java', kt: 'Kotlin', kts: 'Kotlin',
  swift: 'Swift', cs: 'C#', c: 'C', h: 'C Header',
  cpp: 'C++', cc: 'C++', hpp: 'C++ Header', scala: 'Scala',
  dart: 'Dart', ex: 'Elixir', exs: 'Elixir', clj: 'Clojure',
  hs: 'Haskell', ml: 'OCaml', fs: 'F#', erl: 'Erlang',
  lua: 'Lua', r: 'R', R: 'R', jl: 'Julia', pl: 'Perl',
  sh: 'Shell', bash: 'Bash', zsh: 'Zsh', fish: 'Fish',
  bat: 'Batch', cmd: 'Batch', ps1: 'PowerShell',
  sql: 'SQL', graphql: 'GraphQL', gql: 'GraphQL',
  html: 'HTML', htm: 'HTML', css: 'CSS', scss: 'SCSS',
  sass: 'Sass', less: 'Less', xml: 'XML',
  yaml: 'YAML', yml: 'YAML', toml: 'TOML', ini: 'INI',
  json: 'JSON', jsonc: 'JSON', json5: 'JSON5',
  csv: 'CSV', tsv: 'TSV', md: 'Markdown', mdx: 'MDX',
  tf: 'Terraform', hcl: 'HCL', dockerfile: 'Dockerfile',
  makefile: 'Makefile', cmake: 'CMake', proto: 'Protobuf',
  tex: 'LaTeX', rst: 'reStructuredText', org: 'Org',
  txt: 'Plain Text', log: 'Log', conf: 'Config', env: 'Env',
  xlsx: 'Excel', xls: 'Excel', xlsm: 'Excel Macro', xlsb: 'Excel Binary',
  ods: 'OpenDocument Spreadsheet', numbers: 'Numbers',
  doc: 'Word', docx: 'Word', odt: 'OpenDocument', rtf: 'Rich Text',
  pages: 'Pages', ppt: 'PowerPoint', pptx: 'PowerPoint',
  odp: 'OpenDocument Presentation', keynote: 'Keynote', epub: 'EPUB',
  woff: 'WOFF', woff2: 'WOFF2', ttf: 'TrueType', otf: 'OpenType', eot: 'EOT',
  zip: 'ZIP', tar: 'TAR', gz: 'GZip', rar: 'RAR', '7z': '7-Zip',
  iso: 'ISO', dmg: 'DMG', deb: 'DEB', rpm: 'RPM',
  exe: 'Executable', dll: 'DLL', wasm: 'WebAssembly',
  svg: 'SVG', pdf: 'PDF',
}

export function languageLabel(entry) {
  const e = ext(entry)
  return LANG_MAP[e] || e.toUpperCase() || 'File'
}
