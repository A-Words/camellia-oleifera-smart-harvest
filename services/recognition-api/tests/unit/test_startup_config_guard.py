from pathlib import Path

import pytest

from main import _ensure_config_file, _resolve_config_path


def test_resolve_config_path_finds_repo_relative_file_from_service_cwd(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.chdir(Path(__file__).resolve().parents[2])

    path = _resolve_config_path("tooling/config/recognition.yaml")

    assert path.exists()
    assert path.name == "recognition.yaml"


def test_ensure_config_file_raises_with_example_hint(tmp_path: Path) -> None:
    cfg = tmp_path / "recognition.yaml"
    example = tmp_path / "recognition.yaml.example"
    example.write_text("x: 1\n", encoding="utf-8")

    with pytest.raises(RuntimeError) as exc_info:
        _ensure_config_file(cfg, "CAMELLIA_RECOGNITION_CONFIG")

    msg = str(exc_info.value)
    assert "Missing config file" in msg
    assert "CAMELLIA_RECOGNITION_CONFIG" in msg
    assert "recognition.yaml.example" in msg
