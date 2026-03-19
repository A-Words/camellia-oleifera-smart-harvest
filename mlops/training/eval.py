from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
RECOGNITION_API_DIR = REPO_ROOT / 'services' / 'recognition-api'
if str(RECOGNITION_API_DIR) not in sys.path:
    sys.path.insert(0, str(RECOGNITION_API_DIR))

from settings import resolve_torch_device


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Evaluate YOLO model for camellia oleifera fruit detection')
    parser.add_argument('--model', required=True, help='Path to .pt checkpoint')
    parser.add_argument('--data', required=True, help='Path to data YAML')
    parser.add_argument('--imgsz', type=int, default=640)
    parser.add_argument('--device', default='cpu')
    parser.add_argument('--output', default='mlops/artifacts/metrics/eval_metrics.json')
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    resolved_device, device_warning = resolve_torch_device(args.device)
    if device_warning:
        print(f'[device] {device_warning}')

    model_path = str((REPO_ROOT / args.model).resolve() if not Path(args.model).is_absolute() else Path(args.model))
    data_path = str((REPO_ROOT / args.data).resolve() if not Path(args.data).is_absolute() else Path(args.data))
    output_path = (REPO_ROOT / args.output).resolve() if not Path(args.output).is_absolute() else Path(args.output)

    from ultralytics import YOLO

    model = YOLO(model_path)
    metrics = model.val(data=data_path, imgsz=args.imgsz, device=resolved_device)

    out_path = output_path
    out_path.parent.mkdir(parents=True, exist_ok=True)

    payload = {
        'mAP50': float(getattr(metrics.box, 'map50', 0.0)),
        'mAP50_95': float(getattr(metrics.box, 'map', 0.0)),
    }
    out_path.write_text(json.dumps(payload, indent=2), encoding='utf-8')
    print(f'Metrics written to: {out_path}')


if __name__ == '__main__':
    main()
