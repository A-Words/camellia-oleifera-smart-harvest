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
  const lastSubmittedSignature = useState<string>('decision-last-submitted-signature', () => '')

  const hasSnapshot = computed(() => snapshot.value !== null)
  const shouldAutoSubmit = computed(() =>
    Boolean(snapshot.value) &&
    snapshotSignature.value.length > 0 &&
    snapshotSignature.value !== lastSubmittedSignature.value
  )

  function updateSnapshotFromFrame(frame: FrameResult | null, dimensions: SnapshotDimensions) {
    const nextSnapshot = buildDecisionSnapshot(frame, dimensions.width, dimensions.height)
    if (!nextSnapshot) {
      return
    }

    snapshot.value = nextSnapshot
    snapshotSignature.value = createDecisionSnapshotSignature(nextSnapshot)
  }

  function markSubmitted() {
    lastSubmittedSignature.value = snapshotSignature.value
  }

  function clearSnapshot() {
    snapshot.value = null
    snapshotSignature.value = ''
  }

  return {
    snapshot,
    snapshotSignature,
    lastSubmittedSignature,
    hasSnapshot,
    shouldAutoSubmit,
    updateSnapshotFromFrame,
    markSubmitted,
    clearSnapshot
  }
}
