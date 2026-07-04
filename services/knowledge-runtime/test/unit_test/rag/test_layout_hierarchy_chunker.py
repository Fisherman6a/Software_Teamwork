from deepdoc.parser.paddleocr_post_parse import LayoutHierarchyChunker, ValidatedSection


def test_hierarchy_chunker_adds_inherited_context_and_keeps_display_text_clean():
    sections = [
        ValidatedSection(id="h1", text="## 1 Scope", block_type="heading", page_start=1, page_end=1, title="1 Scope", level=2, source_block_ids=["b1"]),
        ValidatedSection(id="h2", text="### 1.1 Safety", block_type="heading", page_start=1, page_end=1, title="1.1 Safety", level=3, source_block_ids=["b2"]),
        ValidatedSection(id="p1", text="The relay must trip within 20 ms.", block_type="paragraph", page_start=1, page_end=1, source_block_ids=["b3"]),
    ]

    chunks = LayoutHierarchyChunker().chunk(sections)

    assert len(chunks) == 1
    assert chunks[0]["content_with_weight"] == "1.1 Safety\nThe relay must trip within 20 ms."
    assert chunks[0]["section_path"] == "1 Scope > 1.1 Safety"
    assert chunks[0]["embedding_text"].startswith("Section: 1 Scope > 1.1 Safety")
    assert chunks[0]["source_block_ids"] == ["b2", "b3"]


def test_hierarchy_chunker_keeps_table_chunks_intact():
    sections = [
        ValidatedSection(id="h1", text="## Tests", block_type="heading", page_start=1, page_end=1, title="Tests", level=2, source_block_ids=["h"]),
        ValidatedSection(
            id="t1",
            text="| Item | Value |\n| --- | --- |\n| Voltage | 220 kV |",
            block_type="table",
            page_start=1,
            page_end=1,
            source_block_ids=["t"],
        ),
    ]

    chunks = LayoutHierarchyChunker().chunk(sections, chunk_token_num=1)

    assert len(chunks) == 1
    assert chunks[0]["doc_type_kwd"] == "table"
    assert "| Voltage | 220 kV |" in chunks[0]["content_with_weight"]


def test_hierarchy_chunker_splits_oversized_paragraph_inside_same_section():
    long_text = "Sentence one. " * 40 + "Sentence two. " * 40
    sections = [
        ValidatedSection(id="h1", text="## Long", block_type="heading", page_start=1, page_end=1, title="Long", level=2, source_block_ids=["h"]),
        ValidatedSection(id="p1", text=long_text, block_type="paragraph", page_start=1, page_end=1, source_block_ids=["p"]),
    ]

    chunks = LayoutHierarchyChunker().chunk(sections, chunk_token_num=20, delimiter=".")

    assert len(chunks) > 1
    assert all(chunk["section_path"] == "Long" for chunk in chunks)
