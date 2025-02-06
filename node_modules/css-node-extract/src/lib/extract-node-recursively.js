import nodeMatchesFilter from './node-matches-filter';

/**
 * Whiteliste a node if it (or one of the nodes parents) matches the given filter.
 */
export default function extractNodeRecursively(node, filterGroups) {
  if (node.parent && node.parent.type !== `root`) {
    return extractNodeRecursively(node.parent, filterGroups);
  }

  let extractNode = false;

  filterGroups.some((filterGroup) => {
    extractNode = filterGroup.filter(filter => !nodeMatchesFilter(node, filter)).length === 0;
    return extractNode;
  });

  return extractNode;
}
