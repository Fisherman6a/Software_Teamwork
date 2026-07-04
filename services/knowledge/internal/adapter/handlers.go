package adapter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func (s *Server) handleListKnowledgeBases(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	page, err := parsePageQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	items, total, err := s.vendor.ListDatasets(r.Context(), s.runtimeScopeID(), page.Page, page.PageSize)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writePageJSON(w, http.StatusOK, knowledgeBasesFromVendor(items), service.Page{
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    total,
	}, reqCtx.RequestID)
}

func (s *Server) handleCreateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body createKnowledgeBaseRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"name": "is required"}))
		return
	}
	parserConfig, err := s.resolveCreateParserConfig(r.Context(), body)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	payload, err := buildCreateDatasetBody(body, parserConfig, createDatasetOptions{
		VendorEmbeddingID: s.cfg.VendorEmbeddingID,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	created, err := s.vendor.CreateDataset(r.Context(), s.runtimeScopeID(), payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusCreated, knowledgeBaseFromVendor(created), reqCtx.RequestID)
}

func (s *Server) resolveCreateParserConfig(ctx context.Context, body createKnowledgeBaseRequest) (map[string]any, error) {
	if body.ChunkStrategy != nil || s.parserConfigs == nil {
		return nil, nil
	}
	snapshot, err := s.parserConfigs.ResolveParserConfig(ctx, "")
	if err != nil {
		return nil, err
	}
	return ragflowParserConfigFromSnapshot(snapshot), nil
}

func (s *Server) resolveUpdateParserConfig(ctx context.Context, body updateKnowledgeBaseRequest) (map[string]any, error) {
	if body.ParserConfigID == nil {
		return nil, nil
	}
	if s.parserConfigs == nil {
		return nil, service.DependencyError("parser config storage is not configured; set DATABASE_URL or KNOWLEDGE_DATABASE_URL", nil)
	}
	snapshot, err := s.parserConfigs.ResolveParserConfigByID(ctx, *body.ParserConfigID)
	if err != nil {
		return nil, err
	}
	return ragflowParserConfigFromSnapshot(snapshot), nil
}

func (s *Server) handleGetKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	ref, err := s.resolveKnowledgeBaseRuntimeRef(r.Context(), r.PathValue("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	dataset, err := s.vendor.GetDataset(r.Context(), s.runtimeScopeID(), ref.ID)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, knowledgeBaseFromVendor(dataset), reqCtx.RequestID)
}

func (s *Server) handleUpdateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body updateKnowledgeBaseRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	parserConfig, err := s.resolveUpdateParserConfig(r.Context(), body)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	payload, err := buildUpdateDatasetBody(body, parserConfig)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	updated, err := s.vendor.UpdateDataset(r.Context(), s.runtimeScopeID(), r.PathValue("knowledgeBaseId"), payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, knowledgeBaseFromVendor(updated), reqCtx.RequestID)
}

func (s *Server) handleDeleteKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.vendor.DeleteDataset(r.Context(), s.runtimeScopeID(), r.PathValue("knowledgeBaseId")); err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	page, err := parsePageQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	ref, err := s.resolveKnowledgeBaseRuntimeRef(r.Context(), r.PathValue("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	items, total, err := s.vendor.ListDocuments(r.Context(), s.runtimeScopeID(), ref.ID, page.Page, page.PageSize)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writePageJSON(w, http.StatusOK, documentsFromVendor(items), service.Page{
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    total,
	}, reqCtx.RequestID)
}

func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	file, header, ok := parseDocumentUpload(w, r, s.maxUploadBytes)
	if !ok {
		return
	}
	defer file.Close()

	contentType := ""
	if header != nil {
		contentType = strings.TrimSpace(header.Header.Get("Content-Type"))
		if inferred := documentContentTypeFromFilename(header.Filename); inferred != "" &&
			(contentType == "" || contentType == genericDocumentContentType) {
			contentType = inferred
		}
	}
	uploaded, err := s.vendor.UploadDocument(r.Context(), s.runtimeScopeID(), r.PathValue("knowledgeBaseId"), header.Filename, contentType, file)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	kbID := r.PathValue("knowledgeBaseId")
	docID := stringField(uploaded, "id")
	if s.cfg.AutoStartIngestion && docID != "" {
		if err := s.vendor.StartDocumentParse(r.Context(), s.runtimeScopeID(), kbID, []string{docID}); err != nil {
			if delErr := s.vendor.DeleteDocument(r.Context(), s.runtimeScopeID(), kbID, docID); delErr != nil {
				s.logger.WarnContext(r.Context(), "upload parse failed and document cleanup failed",
					"service", "knowledge-adapter",
					"request_id", reqCtx.RequestID,
					"document_id", docID,
					"parse_error", err,
					"delete_error", delErr,
				)
			}
			writeAppError(w, r, mapVendorError(err))
			return
		}
		uploaded["run"] = "RUNNING"
	}
	writeJSON(w, http.StatusCreated, documentFromVendor(uploaded), reqCtx.RequestID)
}

func (s *Server) handleUploadDocumentBatch(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	headers, tags, ok := parseDocumentBatchUpload(w, r, documentBatchMaxUploadBytes)
	if !ok {
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	ref, err := s.resolveKnowledgeBaseRuntimeRef(r.Context(), r.PathValue("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if _, err := s.vendor.GetDataset(r.Context(), s.runtimeScopeID(), ref.ID); err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}

	kbID := ref.ID
	results := make([]documentBatchItem, 0, len(headers))
	successCount := 0
	for _, header := range headers {
		result := s.uploadDocumentBatchItem(r.Context(), reqCtx.RequestID, kbID, header, tags)
		if result.Status == "uploaded" {
			successCount++
		}
		results = append(results, result)
	}

	summary := documentBatchSummary{
		TotalCount:   len(results),
		SuccessCount: successCount,
		FailedCount:  len(results) - successCount,
		Results:      results,
	}
	status := http.StatusCreated
	if summary.FailedCount > 0 {
		status = http.StatusMultiStatus
	}
	writeJSON(w, status, summary, reqCtx.RequestID)
}

func (s *Server) uploadDocumentBatchItem(ctx context.Context, requestID, kbID string, header *multipart.FileHeader, tags []string) documentBatchItem {
	filename := ""
	if header != nil {
		filename = header.Filename
	}
	if header == nil {
		return failedDocumentBatchItem(filename, service.ValidationError("request validation failed", map[string]string{"files": "file is required"}))
	}
	if strings.TrimSpace(header.Filename) == "" {
		return failedDocumentBatchItem(filename, service.ValidationError("request validation failed", map[string]string{"files": "filename is required"}))
	}
	if header.Size == 0 {
		return failedDocumentBatchItem(filename, service.ValidationError("request validation failed", map[string]string{"files": "file must not be empty"}))
	}
	if header.Size > defaultMaxUploadBytes {
		return failedDocumentBatchItem(filename, service.ValidationError("request validation failed", map[string]string{"files": "file exceeds maximum upload size"}))
	}

	file, err := header.Open()
	if err != nil {
		return failedDocumentBatchItem(filename, service.NewError(service.CodeInternal, "uploaded file could not be read", err))
	}
	defer file.Close()

	uploaded, err := s.vendor.UploadDocument(ctx, s.runtimeScopeID(), kbID, header.Filename, documentBatchContentType(header), file)
	if err != nil {
		return failedDocumentBatchItem(filename, mapVendorError(err))
	}

	docID := stringField(uploaded, "id")
	if len(tags) > 0 && docID != "" {
		payload, err := buildUpdateDocumentBody(tags)
		if err != nil {
			s.cleanupUploadedBatchDocument(ctx, requestID, kbID, docID, err, "apply_tags")
			return failedDocumentBatchItem(filename, err)
		}
		updated, err := s.vendor.UpdateDocument(ctx, s.runtimeScopeID(), kbID, docID, payload)
		if err != nil {
			mapped := mapVendorError(err)
			s.cleanupUploadedBatchDocument(ctx, requestID, kbID, docID, mapped, "apply_tags")
			return failedDocumentBatchItem(filename, mapped)
		}
		uploaded = documentVendorWithTags(updated, tags)
	}

	if s.cfg.AutoStartIngestion && docID != "" {
		if err := s.vendor.StartDocumentParse(ctx, s.runtimeScopeID(), kbID, []string{docID}); err != nil {
			mapped := mapVendorError(err)
			s.cleanupUploadedBatchDocument(ctx, requestID, kbID, docID, mapped, "start_parse")
			return failedDocumentBatchItem(filename, mapped)
		}
		uploaded["run"] = "RUNNING"
	}

	document := documentFromVendor(uploaded)
	return documentBatchItem{
		Filename: header.Filename,
		Status:   "uploaded",
		Document: &document,
	}
}

func (s *Server) cleanupUploadedBatchDocument(ctx context.Context, requestID, kbID, docID string, cause error, operation string) {
	if strings.TrimSpace(docID) == "" {
		return
	}
	if err := s.vendor.DeleteDocument(ctx, s.runtimeScopeID(), kbID, docID); err != nil {
		s.logger.WarnContext(ctx, "batch upload cleanup failed",
			"service", "knowledge-adapter",
			"request_id", requestID,
			"operation", operation,
			"document_id", docID,
			"cause", cause,
			"delete_error", err,
		)
	}
}

func documentBatchContentType(header *multipart.FileHeader) string {
	if header == nil {
		return ""
	}
	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	mediaType := normalizeDocumentMediaType(contentType)
	if inferred := documentContentTypeFromFilename(header.Filename); inferred != "" &&
		(mediaType == "" || mediaType == genericDocumentContentType) {
		return inferred
	}
	if mediaType != "" {
		return mediaType
	}
	return contentType
}

func failedDocumentBatchItem(filename string, err error) documentBatchItem {
	appErr, ok := service.Classify(err)
	if !ok {
		appErr = service.NewError(service.CodeInternal, "document upload failed", err)
	}
	return documentBatchItem{
		Filename: filename,
		Status:   "failed",
		Error: &documentBatchItemError{
			Code:    appErr.Code,
			Message: documentBatchErrorMessage(appErr),
		},
	}
}

func documentBatchErrorMessage(appErr *service.AppError) string {
	if appErr == nil {
		return "document upload failed"
	}
	if appErr.Code == service.CodeValidation {
		if appErr.Fields != nil {
			if message := strings.TrimSpace(appErr.Fields["files"]); message != "" {
				return message
			}
			if message := strings.TrimSpace(appErr.Fields["file"]); message != "" {
				return message
			}
		}
		return "document upload validation failed"
	}
	switch appErr.Code {
	case service.CodeUnauthorized:
		return "document upload is unauthorized"
	case service.CodeForbidden:
		return "document upload is forbidden"
	case service.CodeNotFound:
		return "related resource was not found"
	case service.CodeConflict:
		return "document upload conflicts with current state"
	case service.CodeRateLimited:
		return "document upload was rate limited"
	case service.CodeDependency:
		return "document upload dependency failed"
	default:
		return "document upload failed"
	}
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	ref, err := s.resolveDocumentRuntimeRef(r.Context(), r.PathValue("documentId"), r.URL.Query().Get("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	doc, err := s.vendor.GetDatasetDocument(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, documentFromVendor(doc), reqCtx.RequestID)
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body updateDocumentRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Tags == nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must include at least one supported field"}))
		return
	}
	kbID, err := requiredDocumentKnowledgeBaseID(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	_, err = s.vendor.GetDatasetDocument(r.Context(), s.runtimeScopeID(), kbID, r.PathValue("documentId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	payload, err := buildUpdateDocumentBody(*body.Tags)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	updated, err := s.vendor.UpdateDocument(r.Context(), s.runtimeScopeID(), kbID, r.PathValue("documentId"), payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	updated = documentVendorWithTags(updated, *body.Tags)
	writeJSON(w, http.StatusOK, documentFromVendor(updated), reqCtx.RequestID)
}

func requiredDocumentKnowledgeBaseID(r *http.Request) (string, error) {
	kbID := strings.TrimSpace(r.URL.Query().Get("knowledgeBaseId"))
	if kbID == "" {
		return "", service.ValidationError("request validation failed", map[string]string{"knowledgeBaseId": "is required"})
	}
	return kbID, nil
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	documentID := r.PathValue("documentId")
	kbID, err := requiredDocumentKnowledgeBaseID(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.vendor.DeleteDocument(r.Context(), s.runtimeScopeID(), kbID, documentID); err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateDocumentDeletionJob(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body createDocumentDeletionJobRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	documentIDs, err := validateDocumentDeletionJobIDs(body.DocumentIDs)
	if err != nil {
		writeAppError(w, r, err)
		return
	}

	ref, err := s.resolveKnowledgeBaseRuntimeRef(r.Context(), r.PathValue("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if _, err := s.vendor.GetDataset(r.Context(), s.runtimeScopeID(), ref.ID); err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}

	results := make([]documentDeletionJobResult, 0, len(documentIDs))
	successCount := 0
	for _, documentID := range documentIDs {
		result := s.deleteDocumentDeletionJobItem(r.Context(), ref.ID, documentID)
		if result.Status == "deleted" {
			successCount++
		}
		results = append(results, result)
	}

	failedCount := len(results) - successCount
	status := "completed"
	httpStatus := http.StatusCreated
	if failedCount > 0 {
		status = "partial_failed"
		httpStatus = http.StatusMultiStatus
	}
	summary := documentDeletionJobSummary{
		ID:              "docdel_" + reqCtx.RequestID,
		Status:          status,
		KnowledgeBaseID: ref.ID,
		TargetIDs:       documentIDs,
		TotalCount:      len(results),
		SuccessCount:    successCount,
		FailedCount:     failedCount,
		Results:         results,
	}
	writeJSON(w, httpStatus, summary, reqCtx.RequestID)
}

func validateDocumentDeletionJobIDs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, service.ValidationError("request validation failed", map[string]string{"documentIds": "is required"})
	}
	if len(raw) > documentDeletionJobMaxDocuments {
		return nil, service.ValidationError("request validation failed", map[string]string{"documentIds": "must include at most 100 documents"})
	}
	seen := make(map[string]struct{}, len(raw))
	documentIDs := make([]string, 0, len(raw))
	for _, value := range raw {
		documentID := strings.TrimSpace(value)
		if documentID == "" {
			return nil, service.ValidationError("request validation failed", map[string]string{"documentIds": "must not include blank ids"})
		}
		if _, ok := seen[documentID]; ok {
			return nil, service.ValidationError("request validation failed", map[string]string{"documentIds": "must not include duplicate ids"})
		}
		seen[documentID] = struct{}{}
		documentIDs = append(documentIDs, documentID)
	}
	return documentIDs, nil
}

func (s *Server) deleteDocumentDeletionJobItem(ctx context.Context, kbID, documentID string) documentDeletionJobResult {
	if err := s.vendor.DeleteDocument(ctx, s.runtimeScopeID(), kbID, documentID); err != nil {
		return failedDocumentDeletionJobResult(documentID, mapVendorError(err))
	}
	return documentDeletionJobResult{
		DocumentID: documentID,
		Status:     "deleted",
	}
}

func failedDocumentDeletionJobResult(documentID string, err error) documentDeletionJobResult {
	appErr, ok := service.Classify(err)
	if !ok {
		appErr = service.NewError(service.CodeInternal, "document delete failed", err)
	}
	return documentDeletionJobResult{
		DocumentID: documentID,
		Status:     "failed",
		Error: &documentDeletionJobItemError{
			Code:    appErr.Code,
			Message: documentDeletionJobErrorMessage(appErr),
		},
	}
}

func documentDeletionJobErrorMessage(appErr *service.AppError) string {
	if appErr == nil {
		return "document delete failed"
	}
	if appErr.Code == service.CodeValidation {
		if appErr.Fields != nil {
			if message := strings.TrimSpace(appErr.Fields["documentIds"]); message != "" {
				return message
			}
			if message := strings.TrimSpace(appErr.Fields["documentId"]); message != "" {
				return message
			}
		}
		return "document delete validation failed"
	}
	switch appErr.Code {
	case service.CodeUnauthorized:
		return "document delete is unauthorized"
	case service.CodeForbidden:
		return "document delete is forbidden"
	case service.CodeNotFound:
		return "document not found"
	case service.CodeConflict:
		return "document delete conflicts with current state"
	case service.CodeRateLimited:
		return "document delete was rate limited"
	case service.CodeDependency:
		return "document delete dependency failed"
	default:
		return "document delete failed"
	}
}

func (s *Server) handleListDocumentChunks(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	page, err := parsePageQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	ref, err := s.resolveDocumentRuntimeRef(r.Context(), r.PathValue("documentId"), r.URL.Query().Get("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if chunkID := strings.TrimSpace(r.URL.Query().Get("id")); chunkID != "" {
		chunk, err := s.vendor.GetChunk(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID, chunkID)
		if err != nil {
			writeAppError(w, r, mapVendorError(err))
			return
		}
		writePageJSON(w, http.StatusOK, []documentChunkSummary{documentChunkFromVendor(chunk, ref.KnowledgeBaseID, ref.ID, 0)}, service.Page{
			Page:     page.Page,
			PageSize: page.PageSize,
			Total:    1,
		}, reqCtx.RequestID)
		return
	}
	_, err = s.vendor.GetDatasetDocument(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	chunks, total, err := s.vendor.ListChunks(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID, page.Page, page.PageSize)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	fallbackStart := (page.Page - 1) * page.PageSize
	writePageJSON(w, http.StatusOK, documentChunksFromVendor(chunks, ref.KnowledgeBaseID, ref.ID, fallbackStart), service.Page{
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    total,
	}, reqCtx.RequestID)
}

func (s *Server) handleGetDocumentContent(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	ref, err := s.resolveDocumentRuntimeRef(r.Context(), r.PathValue("documentId"), r.URL.Query().Get("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	_, err = s.vendor.GetDatasetDocument(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	contentType, body, err := s.vendor.DownloadDocument(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	w.Header().Set("Content-Type", contentType)
	if len(body) > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) handleGetChunk(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	chunkID := strings.TrimSpace(r.PathValue("chunkId"))
	if chunkID == "" {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"chunkId": "is required"}))
		return
	}

	documentID := strings.TrimSpace(r.URL.Query().Get("documentId"))
	if documentID != "" {
		ref, err := s.resolveDocumentRuntimeRef(r.Context(), documentID, r.URL.Query().Get("knowledgeBaseId"))
		if err != nil {
			writeAppError(w, r, err)
			return
		}
		chunk, err := s.vendor.GetChunk(r.Context(), s.runtimeScopeID(), ref.KnowledgeBaseID, ref.ID, chunkID)
		if err != nil {
			writeAppError(w, r, mapVendorError(err))
			return
		}
		writeJSON(w, http.StatusOK, documentChunkFromVendor(chunk, ref.KnowledgeBaseID, ref.ID, 0), reqCtx.RequestID)
		return
	}

	docs, err := s.listRuntimeDocuments(r.Context())
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	for _, doc := range docs {
		docID := strings.TrimSpace(doc.ID)
		kbID := strings.TrimSpace(doc.KnowledgeBaseID)
		if docID == "" || kbID == "" {
			continue
		}
		chunk, err := s.vendor.GetChunk(r.Context(), s.runtimeScopeID(), kbID, docID, chunkID)
		if err == nil {
			writeJSON(w, http.StatusOK, documentChunkFromVendor(chunk, kbID, docID, 0), reqCtx.RequestID)
			return
		}
		if isVendorNotFound(err) {
			continue
		}
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeAppError(w, r, service.NotFoundError("chunk not found"))
}

func (s *Server) handleCreateKnowledgeQuery(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if !trustedProjectRetrievalScope(reqCtx, r) {
		if _, err := readScope(reqCtx); err != nil {
			writeAppError(w, r, err)
			return
		}
	}
	var body knowledgeQueryRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	data, err := s.runKnowledgeQuery(r.Context(), s.runtimeScopeID(), body, retrievalBuildOptions{
		VendorRerankID: s.cfg.VendorRerankID,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	topK := body.TopK
	if topK <= 0 {
		topK = 10
	}
	scoreThreshold := 0.0
	if body.ScoreThreshold != nil {
		scoreThreshold = *body.ScoreThreshold
	}
	writeJSON(w, http.StatusCreated, knowledgeQueryFromVendor(newQueryID(), strings.TrimSpace(body.Query), data, topK, scoreThreshold, body.Rerank, body.RerankTopN, knowledgeQueryTraceOptions{
		VendorEmbeddingID: s.cfg.VendorEmbeddingID,
	}), reqCtx.RequestID)
}

func (s *Server) handleKnowledgeStatistics(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeJSON(w, http.StatusOK, emptyKnowledgeStatisticsSummary(), requestIDFromContext(r.Context()))
		return
	}
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	days, granularity, includeSeries, err := parseKnowledgeStatisticsQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	stats, err := s.collectKnowledgeStatistics(r.Context(), s.runtimeScopeID(), days, granularity, includeSeries)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, stats, reqCtx.RequestID)
}

func (s *Server) collectKnowledgeStatistics(ctx context.Context, runtimeScopeID string, days int, granularity string, includeSeries bool) (knowledgeStatisticsSummary, error) {
	const pageSize = 100
	var datasets []map[string]interface{}
	var kbCount int64
	for page := 1; ; page++ {
		items, total, err := s.vendor.ListDatasets(ctx, runtimeScopeID, page, pageSize)
		if err != nil {
			return emptyKnowledgeStatisticsSummary(), err
		}
		datasets = append(datasets, items...)
		if total > kbCount {
			kbCount = total
		}
		if len(items) == 0 || int64(page*pageSize) >= total {
			break
		}
	}
	if kbCount == 0 {
		kbCount = int64(len(datasets))
	}

	stats := emptyKnowledgeStatisticsSummary()
	stats.KnowledgeBaseCount = kbCount
	datasetSummaries := knowledgeBasesFromVendor(datasets)
	var datasetDocumentCount int64
	var datasetChunkCount int64
	for _, dataset := range datasetSummaries {
		datasetDocumentCount += dataset.DocumentCount
		datasetChunkCount += dataset.ChunkCount
		if includeSeries {
			stats.Series.KnowledgeBaseCount = addKnowledgeStatisticsPoint(stats.Series.KnowledgeBaseCount, dataset.CreatedAt, 1, days, granularity)
		}
	}
	stats.DocumentCount = datasetDocumentCount
	stats.ChunkCount = datasetChunkCount

	if !includeSeries {
		return stats, nil
	}

	var documentCount int64
	var chunkCount int64
	for _, dataset := range datasetSummaries {
		if dataset.ID == "" {
			continue
		}
		var datasetDocumentTotal int64
		var runtimeDocumentTotal int64
		for page := 1; ; page++ {
			items, total, err := s.vendor.ListDocuments(ctx, runtimeScopeID, dataset.ID, page, pageSize)
			if err != nil {
				return emptyKnowledgeStatisticsSummary(), err
			}
			if total > runtimeDocumentTotal {
				runtimeDocumentTotal = total
			}
			documents := documentsFromVendor(items)
			datasetDocumentTotal += int64(len(documents))
			for _, document := range documents {
				chunkCount += document.ChunkCount
				stats.Series.DocumentCount = addKnowledgeStatisticsPoint(stats.Series.DocumentCount, document.CreatedAt, 1, days, granularity)
				stats.Series.ChunkCount = addKnowledgeStatisticsPoint(stats.Series.ChunkCount, document.CreatedAt, document.ChunkCount, days, granularity)
			}
			if len(items) == 0 || int64(page*pageSize) >= total {
				break
			}
		}
		if runtimeDocumentTotal > datasetDocumentTotal {
			datasetDocumentTotal = runtimeDocumentTotal
		}
		documentCount += datasetDocumentTotal
	}
	if documentCount > 0 {
		stats.DocumentCount = documentCount
	}
	if chunkCount > 0 {
		stats.ChunkCount = chunkCount
	}
	return stats, nil
}

func emptyKnowledgeStatisticsSummary() knowledgeStatisticsSummary {
	return knowledgeStatisticsSummary{
		Series: knowledgeStatisticsSeries{
			KnowledgeBaseCount: []knowledgeStatisticsPoint{},
			DocumentCount:      []knowledgeStatisticsPoint{},
			ChunkCount:         []knowledgeStatisticsPoint{},
		},
	}
}

func parseKnowledgeStatisticsQuery(r *http.Request) (int, string, bool, error) {
	query := r.URL.Query()
	includeSeries := query.Has("days") || query.Has("granularity")
	days := 30
	if raw := strings.TrimSpace(query.Get("days")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 90 {
			return 0, "", includeSeries, service.ValidationError("request validation failed", map[string]string{"days": "must be between 1 and 90"})
		}
		days = value
	}
	granularity := strings.TrimSpace(query.Get("granularity"))
	if granularity == "" {
		granularity = "daily"
	}
	if granularity != "daily" && granularity != "hourly" {
		return 0, "", includeSeries, service.ValidationError("request validation failed", map[string]string{"granularity": "must be daily or hourly"})
	}
	return days, granularity, includeSeries, nil
}

func addKnowledgeStatisticsPoint(points []knowledgeStatisticsPoint, value time.Time, count int64, days int, granularity string) []knowledgeStatisticsPoint {
	if value.IsZero() || count <= 0 {
		return points
	}
	value = value.UTC()
	if value.Before(time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)) {
		return points
	}
	bucket := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	if granularity == "hourly" {
		bucket = time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), 0, 0, 0, time.UTC)
	}
	for i := range points {
		if points[i].Date.Equal(bucket) {
			points[i].Count += count
			return points
		}
	}
	return append(points, knowledgeStatisticsPoint{Date: bucket, Count: count})
}

func parsePageQuery(r *http.Request) (service.PageInput, error) {
	page := parsePositiveIntParam(r, "page")
	pageSize := parsePositiveIntParam(r, "pageSize")
	return normalizePage(page, pageSize)
}

func parsePositiveIntParam(r *http.Request, name string) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return value
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, defaultMaxJSONBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		fieldMessage := "must be a valid JSON object"
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			fieldMessage = "exceeds maximum JSON body size"
		}
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": fieldMessage}))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must contain only one JSON object"}))
		return false
	}
	return true
}

func parseDocumentUpload(w http.ResponseWriter, r *http.Request, maxUploadBytes int64) (multipart.File, *multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		fieldMessage := "multipart form is invalid"
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			fieldMessage = "exceeds maximum upload size"
		}
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"file": fieldMessage}))
		return nil, nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"file": "is required"}))
		return nil, nil, false
	}
	if header == nil || header.Size == 0 {
		_ = file.Close()
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"file": "must not be empty"}))
		return nil, nil, false
	}
	return file, header, true
}

func parseDocumentBatchUpload(w http.ResponseWriter, r *http.Request, maxUploadBytes int64) ([]*multipart.FileHeader, []string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(documentBatchMultipartMemoryBytes); err != nil {
		fieldMessage := "multipart form is invalid"
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeAppErrorStatus(w, r, http.StatusRequestEntityTooLarge, service.ValidationError("request validation failed", map[string]string{"files": "exceeds maximum upload size"}))
			return nil, nil, false
		}
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"files": fieldMessage}))
		return nil, nil, false
	}
	if r.MultipartForm == nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"files": "is required"}))
		return nil, nil, false
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"files": "is required"}))
		return nil, nil, false
	}
	if len(files) > documentBatchMaxFiles {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"files": "must include at most 10 files"}))
		return nil, nil, false
	}
	tags, err := parseMultipartTags(r.MultipartForm.Value["tags"])
	if err != nil {
		writeAppError(w, r, err)
		return nil, nil, false
	}
	return files, tags, true
}

func parseMultipartTags(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "[") {
			var decoded []string
			if err := json.Unmarshal([]byte(value), &decoded); err != nil {
				return nil, service.ValidationError("request validation failed", map[string]string{"tags": "must be repeated strings or a JSON string array"})
			}
			for _, tag := range decoded {
				if tag = strings.TrimSpace(tag); tag != "" {
					out = append(out, tag)
				}
			}
			continue
		}
		out = append(out, value)
	}
	return out, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req_fallback"
	}
	return hex.EncodeToString(buf[:])
}

func (s *Server) logRequestFailure(ctx context.Context, requestID, method, path string, status int, durationMs int64) {
	s.logger.ErrorContext(ctx, "http request failed",
		"service", "knowledge-adapter",
		"request_id", requestID,
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", durationMs,
	)
}
