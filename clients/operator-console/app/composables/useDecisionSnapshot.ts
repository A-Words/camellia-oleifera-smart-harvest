import type { DecisionSnapshotRequest } from '~/types/decision'
import type { FrameResult } from '~/types/infer'
import { buildDecisionSnapshot, createDecisionSnapshotSignature } from '~/utils/decision-snapshot'

interface SnapshotDimensions {
  width: number
  height: number
}

export function useDecisionSnapshot() {
  const snapshot = useState<DecisionSnapshotRequest | null>('decision-snapshot', () => null)
  const snapshotSignature = useState<string>('decision-snapshot-signature', () => '')
  const lastArchivedSignature = useState<string>('decision-last-archived-signature', () => '')

  const hasSnapshot = computed(() => snapshot.value !== null)
  const hasPendingObservation = computed(() =>
    Boolean(snapshot.value) &&
    snapshotSignature.value.length > 0 &&
    snapshotSignature.value !== lastArchivedSignature.value
  )

  function updateSnapshotFromFrame(frame: FrameResult | null, dimensions: SnapshotDimensions) {
    const nextSnapshot = buildDecisionSnapshot(frame, dimensions.width, dimensions.height)
    if (!nextSnapshot) {
      return
    }

    snapshot.value = nextSnapshot
    snapshotSignature.value = createDecisionSnapshotSignature(nextSnapshot)
  }

  function markArchived() {
    lastArchivedSignature.value = snapshotSignature.value
  }

  function clearSnapshot() {
    snapshot.value = null
    snapshotSignature.value = ''
    lastArchivedSignature.value = ''
  }

  return {
    snapshot,
    snapshotSignature,
    lastArchivedSignature,
    hasSnapshot,
    hasPendingObservation,
    updateSnapshotFromFrame,
    markArchived,
    clearSnapshot
  }
}
