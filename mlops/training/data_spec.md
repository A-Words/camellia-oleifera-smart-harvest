# Data Specification

## Label classes
- `0`: camellia_oleifera_fruit

## Annotation rules
- Every visible camellia oleifera fruit should have one bounding box.
- Each bounding box must use the single detection class `camellia_oleifera_fruit`.
- Occluded camellia oleifera fruits over 60% should be ignored.
- Blurry objects smaller than 12x12 pixels should be ignored.

## Split strategy
- Train/Val/Test = 70/15/15
- Keep scene-level split to avoid leakage from adjacent frames.
- Freeze a `golden_test_set` that is never used in training.
- Current baseline dataset layout follows `images/{train,val,test}` and `labels/{train,val,test}`.
- Current source dataset counts are `train=1012`, `val=337`, `test=328`.

## Quality gates
- 5% random double-annotation sample for consistency check.
- Generate detection count report each data refresh.
- Maintain bad-case list for hard examples (low light, blur, cluster occlusion).
