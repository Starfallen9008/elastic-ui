import test from 'node:test';
import assert from 'node:assert/strict';
import { processGraph } from '../src/lib/processGraph.mjs';

test('graph preserves forks, deduplicates nodes and places parents before children', () => {
  const graph = processGraph({ nodes: [{entityId:'root'}, {entityId:'alert', parentEntityId:'root', isAlert:true}], children: [{entityId:'a',parentEntityId:'alert'}, {entityId:'b',parentEntityId:'alert'}, {entityId:'a'}] });
  assert.equal(graph.nodes.length, 4);
  assert.equal(graph.edges.length, 3);
  const [root, alert, a, b] = graph.nodes;
  assert.ok(root.x < alert.x && alert.x < a.x);
  assert.equal(a.x, b.x);
  assert.notEqual(a.y, b.y);
  assert.equal(alert.y, (a.y + b.y) / 2);
});

test('missing records, disconnected roots and malformed cycles cannot break layout', () => {
  const graph = processGraph({ nodes: [{entityId:'a', parentEntityId:'b'}, {entityId:'b',parentEntityId:'a'}, {entityId:'c',parentEntityId:'missing'}] });
  assert.equal(graph.nodes.length, 3);
  assert.ok(graph.nodes.every(node => Number.isFinite(node.x) && Number.isFinite(node.y)));
  assert.equal(processGraph(null).nodes.length, 0);
});
