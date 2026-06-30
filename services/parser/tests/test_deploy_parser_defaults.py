from pathlib import Path


def test_deploy_defaults_enable_ppstructurev3_backend():
    repo_root = Path(__file__).resolve().parents[3]
    env_example = (repo_root / "deploy/.env.example").read_text(encoding="utf-8")
    compose = (repo_root / "deploy/docker-compose.yml").read_text(encoding="utf-8")

    assert "PARSER_BACKEND=ppstructurev3" in env_example
    assert "PARSER_BACKEND: ${PARSER_BACKEND:-ppstructurev3}" in compose
    assert "PARSER_BACKEND=document" not in env_example
    assert "PARSER_BACKEND: ${PARSER_BACKEND:-document}" not in compose
