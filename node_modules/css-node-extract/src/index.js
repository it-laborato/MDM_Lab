import postcss from 'postcss';

import postcssNodeExtract from './lib/postcss-node-extract';

import { PRESERVE_LINES_END, PRESERVE_LINES_START } from './const';

/**
 * Synchronously extract nodes from a string.
 */
export const processSync = ({
  css,
  filters,
  customFilters,
  postcssSyntax,
  preserveLines,
}) => postcss(postcssNodeExtract(filters, customFilters, preserveLines))
  .process(css, { syntax: postcssSyntax }).css
  .replace(new RegExp(`\\/\\* ${PRESERVE_LINES_START}|${PRESERVE_LINES_END} \\*\\/`, `g`), ``);

/**
 * Asynchronously extract nodes from a string.
 */
export const process = options => new Promise((resolve) => {
  const result = processSync(options);
  resolve(result);
});
