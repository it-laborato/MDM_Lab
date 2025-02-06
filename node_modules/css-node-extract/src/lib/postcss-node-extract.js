import { camelCase } from 'change-case';
import postcss from 'postcss';

import extractNodeRecursively from './extract-node-recursively';
import filterDefinitions from './filter-definitions';

import { PRESERVE_LINES_END, PRESERVE_LINES_START } from '../const';

/**
 * A PostCSS plugin for extracting nodes from CSS code.
 */
export default function postcssNodeExtract(
  filterNames,
  customFilters,
  preserveLines,
) {
  const filterNamesArray = Array.isArray(filterNames) ? filterNames : [filterNames];
  Object.assign(filterDefinitions, customFilters);

  return postcss.plugin(`postcss-node-extract`, () => (nodes) => {
    nodes.walk((rule) => {
      let filterRule = false;
      filterNamesArray.some((filterName) => {
        const filterNameCamelCase = camelCase(filterName);
        filterRule = extractNodeRecursively(rule, filterDefinitions[filterNameCamelCase]);
        return filterRule;
      });
      if (!filterRule) {
        if (preserveLines) {
          const ruleLines = rule.toString().split(/\r\n|\r|\n/).length;

          rule.cloneBefore({
            type: `comment`,
            text: `${PRESERVE_LINES_START}${`\n`.repeat(ruleLines - 1)}${PRESERVE_LINES_END}`,
            raws: Object.assign(rule.raws, { left: ` `, right: ` ` }),
          });
        }

        rule.remove();
      }
    });
  });
}
