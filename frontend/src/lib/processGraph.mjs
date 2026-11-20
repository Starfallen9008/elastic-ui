// Forest layout: siblings share a column; parents are centered over their leaves.
/** @typedef {import('../../wailsjs/go/models').service.ProcessNode & {x:number, y:number, children:GraphNode[], parent:GraphNode|null}} GraphNode */
/** @param {import('../../wailsjs/go/models').service.ProcessChain|null} chain */
export function processGraph(chain) {
  /** @type {GraphNode[]} */
  const nodes = [];
  /** @type {Map<string, GraphNode>} */
  const byID = new Map();
  for (const node of [...(chain?.nodes || []), ...(chain?.children || [])]) {
    if (!node.entityId || byID.has(node.entityId)) continue;
    const item = { ...node, x: 0, y: 0, children: [], parent: null };
    nodes.push(item); byID.set(item.entityId, item);
  }
  const lineage = chain?.nodes || [];
  for (const node of nodes) {
    let parent = byID.get(node.parentEntityId);
    const index = lineage.findIndex(item => item.entityId === node.entityId);
    if (!node.parentEntityId && index > 0) parent = byID.get(lineage[index - 1].entityId);
    if (parent === node) continue;
    let cursor = parent || null, cyclic = false;
    while (cursor) { if (cursor === node) { cyclic = true; break; } cursor = cursor.parent; }
    if (parent && !cyclic) { node.parent = parent; parent.children.push(node); }
  }
  let row = 0;
  /** @param {GraphNode} node @param {number} depth */
  function place(node, depth) {
    node.x = 48 + depth * 260;
    if (!node.children.length) node.y = 50 + row++ * 110;
    else { node.children.forEach(child => place(child, depth + 1)); node.y = (node.children[0].y + node.children[node.children.length - 1].y) / 2; }
  }
  nodes.filter(node => !node.parent).forEach(node => place(node, 0));
  const edges = nodes.flatMap(node => {
    if (!node.parent) return [];
    const x = node.parent.x + 208, y = node.parent.y + 36, endX = node.x, endY = node.y + 36;
    return [{ id: node.entityId, alert: node.isAlert, path: `M ${x} ${y} C ${x + 26} ${y}, ${endX - 26} ${endY}, ${endX} ${endY}` }];
  });
  return { nodes, edges, width: Math.max(400, ...nodes.map(n => n.x + 256)), height: Math.max(220, ...nodes.map(n => n.y + 122)) };
}
