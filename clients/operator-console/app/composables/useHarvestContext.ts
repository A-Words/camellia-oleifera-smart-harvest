export function useHarvestContext() {
  const selectedPlotId = useState<string>('harvest-selected-plot-id', () => '')
  const selectedTreeId = useState<string>('harvest-selected-tree-id', () => '')

  function setSelectedPlot(plotId: string) {
    if (selectedPlotId.value !== plotId) {
      selectedPlotId.value = plotId
      selectedTreeId.value = ''
      return
    }
    selectedPlotId.value = plotId
  }

  function setSelectedTree(treeId: string) {
    selectedTreeId.value = treeId
  }

  return {
    selectedPlotId,
    selectedTreeId,
    setSelectedPlot,
    setSelectedTree
  }
}
