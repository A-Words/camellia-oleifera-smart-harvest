from pathlib import Path

from core.recognition.adapters.yolo_stable import YoloStableAdapter
from settings import ModelConfig, load_model_config


def test_model_source_uses_explicit_path() -> None:
    cfg = ModelConfig(yolo_version="yolo26n", model_path="weights/custom.pt")
    adapter = YoloStableAdapter(cfg)
    assert adapter._resolve_model_source() == "weights/custom.pt"


def test_model_source_falls_back_to_yolo_version() -> None:
    cfg = ModelConfig(yolo_version="yolo26n", model_path="")
    adapter = YoloStableAdapter(cfg)
    assert adapter._resolve_model_source() == "yolo26n.pt"


def test_load_model_config_resolves_repo_relative_model_path_from_service_cwd(
    monkeypatch,
    tmp_path: Path,
) -> None:
    monkeypatch.chdir(Path(__file__).resolve().parents[2])
    cfg_path = tmp_path / "recognition.yaml"
    cfg_path.write_text(
        '\n'.join(
            [
                'yolo_version: "yolo26n"',
                'model_version: "test"',
                'model_path: "yolo26n.pt"',
                'device: "cpu"',
            ]
        ),
        encoding="utf-8",
    )

    cfg = load_model_config(cfg_path)

    resolved = Path(cfg.model_path)
    assert resolved.is_absolute()
    assert resolved.exists()
    assert resolved.name == "yolo26n.pt"
