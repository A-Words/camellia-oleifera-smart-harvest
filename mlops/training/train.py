from __future__ import annotations

import argparse
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
RECOGNITION_API_DIR = REPO_ROOT / 'services' / 'recognition-api'
if str(RECOGNITION_API_DIR) not in sys.path:
    sys.path.insert(0, str(RECOGNITION_API_DIR))

from settings import resolve_torch_device


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Train YOLO baseline for camellia oleifera fruit detection')
    parser.add_argument('--data', required=True, help='Path to data YAML')
    parser.add_argument('--model', default='yolo26n.pt', help='Pretrained model checkpoint')
    parser.add_argument('--epochs', type=int, default=100)
    parser.add_argument('--imgsz', type=int, default=640)
    parser.add_argument('--batch', type=int, default=16)
    parser.add_argument('--device', default='cpu')
    parser.add_argument('--project', default='mlops/artifacts/models')
    parser.add_argument('--name', default='camellia_detection_v1')
    parser.add_argument('--export-onnx', action='store_true')
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    resolved_device, device_warning = resolve_torch_device(args.device)
    if device_warning:
        print(f'[device] {device_warning}')

    data_path = str((REPO_ROOT / args.data).resolve() if not Path(args.data).is_absolute() else Path(args.data))
    model_path = str((REPO_ROOT / args.model).resolve() if not Path(args.model).is_absolute() else Path(args.model))
    project_path = str((REPO_ROOT / args.project).resolve() if not Path(args.project).is_absolute() else Path(args.project))

    from ultralytics import YOLO

    model = YOLO(model_path)
    results = model.train(
        data=data_path,
        epochs=args.epochs,
        imgsz=args.imgsz,
        batch=args.batch,
        device=resolved_device,
        project=project_path,
        name=args.name,
    )

    save_dir = Path(results.save_dir)
    print(f'Training done: {save_dir}')

    best_pt = save_dir / 'weights' / 'best.pt'
    if best_pt.exists():
        print(f'Best checkpoint: {best_pt}')

    if args.export_onnx and best_pt.exists():
        export_model = YOLO(str(best_pt))
        export_path = export_model.export(format='onnx')
        print(f'ONNX exported: {export_path}')


if __name__ == '__main__':
    main()
