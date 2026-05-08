import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import css from 'highlight.js/lib/languages/css'
import diff from 'highlight.js/lib/languages/diff'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import go from 'highlight.js/lib/languages/go'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import markdown from 'highlight.js/lib/languages/markdown'
import python from 'highlight.js/lib/languages/python'
import sql from 'highlight.js/lib/languages/sql'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import yaml from 'highlight.js/lib/languages/yaml'

const languages = {
  bash,
  sh: bash,
  shell: bash,
  css,
  diff,
  dockerfile,
  go,
  golang: go,
  java,
  javascript,
  js: javascript,
  json,
  markdown,
  md: markdown,
  python,
  py: python,
  sql,
  typescript,
  ts: typescript,
  html: xml,
  xml,
  vue: xml,
  yaml,
  yml: yaml
}

Object.entries(languages).forEach(([name, language]) => {
  if (!hljs.getLanguage(name)) {
    hljs.registerLanguage(name, language)
  }
})

export default hljs
