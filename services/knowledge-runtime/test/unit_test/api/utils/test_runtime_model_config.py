import pytest

from api.utils import runtime_model_config as runtime_model_config_module
from api.utils.runtime_model_config import default_model_id, runtime_model_config
from common.constants import LLMType


MODEL_ENV_KEYS = [
    "KNOWLEDGE_RUNTIME_EMBEDDING_FACTORY",
    "KNOWLEDGE_RUNTIME_EMBEDDING_MODEL",
    "KNOWLEDGE_RUNTIME_EMBEDDING_BASE_URL",
	"KNOWLEDGE_RUNTIME_RERANK_FACTORY",
	"KNOWLEDGE_RUNTIME_RERANK_MODEL",
	"KNOWLEDGE_RUNTIME_RERANK_BASE_URL",
    "KNOWLEDGE_RUNTIME_CHAT_FACTORY",
    "KNOWLEDGE_RUNTIME_CHAT_MODEL",
    "KNOWLEDGE_RUNTIME_CHAT_BASE_URL",
]


@pytest.fixture(autouse=True)
def clear_model_env(monkeypatch):
    for key in MODEL_ENV_KEYS:
        monkeypatch.delenv(key, raising=False)


def test_runtime_model_config_accepts_ai_gateway_embedding(monkeypatch):
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_EMBEDDING_FACTORY", "AI_GATEWAY")
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_EMBEDDING_MODEL", "BAAI/bge-m3")
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_EMBEDDING_BASE_URL", "http://127.0.0.1:8086/internal/v1")

    config = runtime_model_config(LLMType.EMBEDDING)

    assert config["llm_factory"] == "AI_GATEWAY"
    assert config["api_key"] == ""
    assert config["llm_name"] == "BAAI/bge-m3"
    assert config["api_base"] == "http://127.0.0.1:8086/internal/v1"


def test_runtime_model_config_rejects_direct_embedding_factory(monkeypatch):
	monkeypatch.setenv("KNOWLEDGE_RUNTIME_EMBEDDING_FACTORY", "SILICONFLOW")
	monkeypatch.setenv("KNOWLEDGE_RUNTIME_EMBEDDING_MODEL", "BAAI/bge-m3")

	with pytest.raises(LookupError, match="must use AI_GATEWAY"):
		runtime_model_config(LLMType.EMBEDDING)


def test_default_model_id_rejects_direct_rerank_reference(monkeypatch):
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_RERANK_MODEL", "BAAI/bge-reranker-v2-m3@default@SILICONFLOW")

    with pytest.raises(LookupError, match="must use AI_GATEWAY"):
        default_model_id(LLMType.RERANK)


def test_runtime_model_config_accepts_ai_gateway_chat_with_string_chat_cfg(monkeypatch):
    monkeypatch.setattr(runtime_model_config_module.settings, "CHAT_CFG", "", raising=False)

    config = runtime_model_config(LLMType.CHAT, "deepseek-ai/DeepSeek-V4-Flash@AI_GATEWAY")

    assert config["llm_factory"] == "AI_GATEWAY"
    assert config["api_key"] == ""
    assert config["llm_name"] == "deepseek-ai/DeepSeek-V4-Flash"
    assert config["api_base"] == "http://127.0.0.1:8086/internal/v1"
