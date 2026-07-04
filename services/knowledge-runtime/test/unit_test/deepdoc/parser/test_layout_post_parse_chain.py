from deepdoc.parser.paddleocr_adapter import PaddleOCRResultAdapter
from deepdoc.parser.paddleocr_post_parse import (
    FakeLayoutRepairer,
    LayoutQualityGate,
    PaddleOCRPostParseChain,
    ValidatedSection,
)


def _adapt(raw):
    return PaddleOCRResultAdapter().adapt(raw)


def test_adapter_emits_stable_canonical_block_ids():
    pages = _adapt(
        {
            "layoutParsingResults": [
                {
                    "prunedResult": {
                        "parsing_res_list": [
                            {
                                "block_label": "paragraph_title",
                                "block_content": "1 Scope",
                                "block_bbox": [100, 20, 300, 40],
                                "block_order": "7",
                                "block_id": "provider-7",
                            }
                        ]
                    }
                }
            ]
        }
    )

    block = pages[0].blocks[0]
    assert block.id == "p1-b0000"
    assert block.order == 7
    assert block.metadata["canonical_block_id"] == "p1-b0000"
    assert block.metadata["source_block_id"] == "provider-7"


def test_quality_gate_marks_core_dirty_signals():
    gate = LayoutQualityGate()
    sections = [
        ValidatedSection(id="s1", text="abc �� \x01 ??? !!! ### |||", block_type="paragraph", page_start=1, page_end=1),
        ValidatedSection(
            id="s2",
            text="A    B    C\n1    2    3",
            block_type="paragraph",
            page_start=1,
            page_end=1,
        ),
        ValidatedSection(id="s3", text="   ", block_type="paragraph", page_start=1, page_end=1),
    ]

    decisions = gate.classify(sections)

    assert "garbled_text" in decisions[0][1]
    assert "table_shape_suspicious" in decisions[1][1]
    assert "low_information" in decisions[2][1]


def test_quality_gate_marks_fragmented_headings():
    sections = [
        ValidatedSection(id="s1", text="## A", block_type="heading", page_start=1, page_end=1, title="A", level=2),
        ValidatedSection(id="s2", text="## B", block_type="heading", page_start=1, page_end=1, title="B", level=2),
        ValidatedSection(id="s3", text="## C", block_type="heading", page_start=1, page_end=1, title="C", level=2),
    ]

    decisions = LayoutQualityGate().classify(sections)

    assert all("fragmented_heading" in flags for _, flags in decisions)


def test_valid_llm_repair_is_accepted_after_fidelity_validation():
    raw = {
        "layoutParsingResults": [
            {
                "prunedResult": {
                    "parsing_res_list": [
                        {
                            "block_label": "table",
                            "block_content": "<table><tr><td>A</td><td>B</td></tr><tr><td>1 2</td></tr></table>",
                            "block_order": 1,
                        }
                    ]
                }
            }
        ]
    }
    repairer = FakeLayoutRepairer(
        {
            "dirty-0000": [
                {
                    "text": "| A | B |\n| --- | --- |\n| 1 | 2 |",
                    "block_type": "table",
                    "source_block_ids": ["p1-b0000"],
                }
            ]
        }
    )

    result = PaddleOCRPostParseChain(repairer=repairer).run(_adapt(raw))

    assert result.sections[0].repair_status == "repaired"
    assert result.sections[0].text == "| A | B |\n| --- | --- |\n| 1 | 2 |"


def test_invalid_llm_repair_falls_back_to_deterministic_output():
    raw = {
        "layoutParsingResults": [
            {
                "prunedResult": {
                    "parsing_res_list": [
                        {
                            "block_label": "paragraph",
                            "block_content": "Asset ID EQ-17 was checked on 2026-07-04.",
                            "block_order": 1,
                        }
                    ]
                }
            }
        ]
    }
    repairer = FakeLayoutRepairer(
        {
            "dirty-0000": [
                {
                    "text": "Asset ID EQ-99 was checked on 2026-07-05.",
                    "block_type": "paragraph",
                    "source_block_ids": ["p1-b0000"],
                }
            ]
        }
    )
    chain = PaddleOCRPostParseChain(repairer=repairer)
    sections = [ValidatedSection.from_semantic(section, idx) for idx, section in enumerate(chain.normalizer.normalize(_adapt(raw)))]
    dirty = [(sections[0], ["garbled_text"])]
    window = chain._dirty_windows(dirty)[0]

    repaired = chain._repair_or_fallback(window)

    assert repaired[0].repair_status == "repair_rejected"
    assert "EQ-17" in repaired[0].text
