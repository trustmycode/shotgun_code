import { reactive } from 'vue';

export function useFileTreeUtils({ useGitignore, useCustomIgnore, manuallyToggledNodes, fileTree, addLog }) {
  function calculateNodeExcludedState(node) {
    const manualToggle = manuallyToggledNodes.get(node.relPath);
    if (manualToggle !== undefined) return manualToggle;
    if (useGitignore.value && node.isGitignored) return true;
    if (useCustomIgnore.value && node.isCustomIgnored) return true;
    return false;
  }

  function mapDataToTreeRecursive(nodes, parent) {
    if (!nodes) return [];
    return nodes.map((node) => {
      const isRootNode = parent === null;
      const reactiveNode = reactive({
        ...node,
        expanded: node.isDir ? isRootNode : undefined,
        parent,
        children: [],
      });
      reactiveNode.excluded = calculateNodeExcludedState(reactiveNode);

      if (node.children && node.children.length > 0) {
        reactiveNode.children = mapDataToTreeRecursive(node.children, reactiveNode);
      }
      return reactiveNode;
    });
  }

  function isAnyParentVisuallyExcluded(node) {
    if (!node || !node.parent) {
      return false;
    }
    let current = node.parent;
    while (current) {
      if (current.excluded) {
        return true;
      }
      current = current.parent;
    }
    return false;
  }

  function hasVisuallyIncludedDescendant(node) {
    if (!node || !node.children || node.children.length === 0) {
      return false;
    }
    return node.children.some((child) => !child.excluded || hasVisuallyIncludedDescendant(child));
  }

  function collectTrulyExcludedPaths(nodes, target) {
    if (!nodes || nodes.length === 0) return;
    nodes.forEach((node) => {
      if (node.excluded && !hasVisuallyIncludedDescendant(node)) {
        target.push(node.relPath);
      } else if (node.children && node.children.length > 0) {
        collectTrulyExcludedPaths(node.children, target);
      }
    });
  }

  function buildExcludedPathsPayload() {
    const excluded = [];
    collectTrulyExcludedPaths(fileTree.value, excluded);
    return excluded;
  }

  function collectIgnoredPathsOnly(nodes, target) {
    if (!nodes || nodes.length === 0) return;
    nodes.forEach((node) => {
      const ignoredByGit = useGitignore.value && node.isGitignored;
      const ignoredByCustom = useCustomIgnore.value && node.isCustomIgnored;
      const ignoredByRules = ignoredByGit || ignoredByCustom;

      if (ignoredByRules && node.relPath) {
        target.push(node.relPath);
        return;
      }

      if (node.children && node.children.length > 0) {
        collectIgnoredPathsOnly(node.children, target);
      }
    });
  }

  function buildIgnoredPathsPayloadForAutoContext() {
    const ignored = [];
    collectIgnoredPathsOnly(fileTree.value, ignored);
    return ignored;
  }

  function normalizeRelPath(value) {
    if (!value) return '';
    return value.replace(/\\/g, '/').replace(/^\.\//, '').replace(/^\/+/, '');
  }

  function clearDescendantManualToggles(node) {
    if (node.children && node.children.length > 0) {
      node.children.forEach((child) => {
        manuallyToggledNodes.delete(child.relPath);
        clearDescendantManualToggles(child);
      });
    }
  }

  function toggleExcludeNode(nodeToToggle) {
    if (isAnyParentVisuallyExcluded(nodeToToggle) && nodeToToggle.excluded) {
      nodeToToggle.excluded = false;
    } else {
      nodeToToggle.excluded = !nodeToToggle.excluded;
    }
    manuallyToggledNodes.set(nodeToToggle.relPath, nodeToToggle.excluded);

    if (nodeToToggle.isDir) {
      clearDescendantManualToggles(nodeToToggle);
    }

    addLog(`Toggled exclusion for ${nodeToToggle.name} to ${nodeToToggle.excluded}`, 'info', 'bottom');
  }

  function updateAllNodesExcludedState(nodesToUpdate) {
    _updateAllNodesExcludedStateRecursive(nodesToUpdate, false);
  }

  function _updateAllNodesExcludedStateRecursive(nodesToUpdate, parentIsVisuallyExcluded) {
    if (!nodesToUpdate || nodesToUpdate.length === 0) return;
    nodesToUpdate.forEach((node) => {
      const manualToggle = manuallyToggledNodes.get(node.relPath);
      let isExcludedByRule = false;
      if (useGitignore.value && node.isGitignored) isExcludedByRule = true;
      if (useCustomIgnore.value && node.isCustomIgnored) isExcludedByRule = true;

      if (manualToggle !== undefined) {
        node.excluded = manualToggle;
      } else {
        node.excluded = isExcludedByRule || parentIsVisuallyExcluded;
      }

      if (node.children && node.children.length > 0) {
        _updateAllNodesExcludedStateRecursive(node.children, node.excluded);
      }
    });
  }

  return {
    mapDataToTreeRecursive,
    toggleExcludeNode,
    updateAllNodesExcludedState,
    buildExcludedPathsPayload,
    buildIgnoredPathsPayloadForAutoContext,
    normalizeRelPath,
  };
}
