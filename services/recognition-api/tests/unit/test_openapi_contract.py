from __future__ import annotations

from pathlib import Path

import yaml

OPENAPI_PATH = Path("shared/contracts/openapi.yaml")
PRD_PATH = Path("docs/prd.md")


def _load_openapi() -> dict:
    return yaml.safe_load(OPENAPI_PATH.read_text(encoding="utf-8"))


def _collect_refs(node: object, out: list[str]) -> None:
    if isinstance(node, dict):
        ref = node.get("$ref")
        if isinstance(ref, str):
            out.append(ref)
        for value in node.values():
            _collect_refs(value, out)
        return
    if isinstance(node, list):
        for item in node:
            _collect_refs(item, out)


def test_openapi_yaml_is_parsable() -> None:
    doc = _load_openapi()
    assert isinstance(doc, dict)
    assert doc.get("openapi") == "3.1.0"


def test_openapi_refs_have_no_broken_links() -> None:
    doc = _load_openapi()
    refs: list[str] = []
    _collect_refs(doc, refs)

    components = doc.get("components", {})
    schemas = components.get("schemas", {})
    parameters = components.get("parameters", {})

    missing: list[str] = []
    for ref in refs:
        if ref.startswith("#/components/schemas/"):
            name = ref.split("/")[-1]
            if name not in schemas:
                missing.append(ref)
        elif ref.startswith("#/components/parameters/"):
            name = ref.split("/")[-1]
            if name not in parameters:
                missing.append(ref)

    assert not missing


def test_new_paths_exist_and_operation_ids_are_unique() -> None:
    doc = _load_openapi()
    paths = doc["paths"]

    assert "/healthz" in paths
    assert "/v1/health" in paths
    assert "/v1/recognition/image" in paths
    assert "/v1/recognition/stream" in paths

    operation_ids: list[str] = []
    for methods in paths.values():
        for op in methods.values():
            if isinstance(op, dict) and "operationId" in op:
                operation_ids.append(op["operationId"])
    assert len(operation_ids) == len(set(operation_ids))


def test_recognition_image_response_codes_and_security() -> None:
    doc = _load_openapi()
    operation = doc["paths"]["/v1/recognition/image"]["post"]
    responses = operation["responses"]

    assert {"200", "400", "401", "403", "413", "503"}.issubset(set(responses.keys()))
    assert {"ApiKeyAuth": []} in operation["security"]


def test_system_health_endpoints_are_public() -> None:
    doc = _load_openapi()
    assert doc["paths"]["/healthz"]["get"]["security"] == []
    assert doc["paths"]["/v1/health"]["get"]["security"] == []


def test_recognition_stream_declares_auth() -> None:
    doc = _load_openapi()
    operation = doc["paths"]["/v1/recognition/image"]["post"]
    assert {"ApiKeyAuth": []} in operation["security"]


def test_prd_field_names_match_contract() -> None:
    doc = _load_openapi()
    prd_text = PRD_PATH.read_text(encoding="utf-8")

    required_tokens = [
        "识别结果域",
        "作业决策域",
        "作业管理域",
        "/v1/recognition/image",
        "/v1/recognition/stream",
    ]
    for token in required_tokens:
        assert token in prd_text

    detection_props = doc["components"]["schemas"]["Detection"]["properties"]
    frame_props = doc["components"]["schemas"]["FrameResult"]["properties"]
    session_props = doc["components"]["schemas"]["SessionSummary"]["properties"]
    model_meta_props = doc["components"]["schemas"]["ModelMeta"]["properties"]

    assert detection_props["class_name"]["enum"] == ["camellia_oleifera_fruit"]
    assert detection_props["ripeness"]["enum"] == ["harvestable", "not_ready", "occluded_unclear"]
    assert "detections" in frame_props
    assert "frame_summary" in frame_props
    assert list(session_props.keys()) == ["total_detected"]
    assert "load_error" in model_meta_props
    assert "ripeness_enabled" in model_meta_props
    assert "ripeness_loaded" in model_meta_props
