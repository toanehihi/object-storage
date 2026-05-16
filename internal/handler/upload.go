package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/toanehihi/object-storage/internal/middleware"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/internal/service"
	"github.com/toanehihi/object-storage/internal/utils/response"
)

type ChunkUploadedResponse struct {
	FileID     uuid.UUID `json:"fileId"`
	ChunkIndex int32     `json:"chunkIndex"`
	Status     string    `json:"status"`
}

type CompleteUploadResponse struct {
	FileID uuid.UUID `json:"fileId"`
	Status string    `json:"status"`
}

type UploadHandler struct {
	svc *service.UploadService
}

func NewUploadHandler(svc *service.UploadService) *UploadHandler {
	return &UploadHandler{svc: svc}
}

func (h *UploadHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.initUpload)
	g.GET("/:fileId/chunks/:chunkIndex/url", h.getChunkUploadURL)
	g.PUT("/:fileId/chunks/:chunkIndex", h.chunkComplete)
	g.GET("/:fileId/status", h.status)
	g.POST("/:fileId/complete", h.complete)
}

func (h *UploadHandler) initUpload(c echo.Context) error {
	ownerID := middleware.UserIDFromContext(c)

	var req service.InitUploadRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Filename == "" || req.Size <= 0 || req.ChunkSize <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "filename, size, and chunkSize are required")
	}

	resp, err := h.svc.InitUpload(c.Request().Context(), ownerID, req)
	if errors.Is(err, service.ErrInvalidChunkSize) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err != nil {
		c.Logger().Errorf("Failed to init upload: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to init upload: "+err.Error())
	}

	return c.JSON(http.StatusCreated, response.OK(resp, "upload initialized"))
}

func (h *UploadHandler) getChunkUploadURL(c echo.Context) error {
	ownerID := middleware.UserIDFromContext(c)

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	chunkIndex, err := strconv.Atoi(c.Param("chunkIndex"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid chunk index")
	}

	resp, err := h.svc.GetChunkUploadURL(c.Request().Context(), ownerID, fileID, chunkIndex)
	if errors.Is(err, service.ErrFileNotOwned) {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	if errors.Is(err, service.ErrChunkIndexOutOfRange) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get chunk upload url")
	}

	return c.JSON(http.StatusOK, response.OK(resp, ""))
}

func (h *UploadHandler) chunkComplete(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	var body struct {
		ChunkIndex int32  `json:"chunkIndex"`
		ETag       string `json:"etag"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	req := service.ChunkCompleteRequest{
		FileID:     fileID,
		ChunkIndex: body.ChunkIndex,
		ETag:       body.ETag,
	}

	if err := h.svc.MarkChunkComplete(c.Request().Context(), req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark chunk")
	}

	return c.JSON(http.StatusOK, response.OK(ChunkUploadedResponse{
		FileID:     fileID,
		ChunkIndex: body.ChunkIndex,
		Status:     "uploaded",
	}, "chunk marked as uploaded"))
}

func (h *UploadHandler) status(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	resp, err := h.svc.GetStatus(c.Request().Context(), fileID)
	if errors.Is(err, repository.ErrFileNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get status")
	}

	return c.JSON(http.StatusOK, response.OK(resp, ""))
}

func (h *UploadHandler) complete(c echo.Context) error {
	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	if err := h.svc.CompleteUpload(c.Request().Context(), fileID); err != nil {
		if errors.Is(err, service.ErrUploadIncomplete) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		c.Logger().Errorf("failed to complete upload: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to complete upload: " + err.Error())
	}

	// Fetch updated file to return correct status
	statusResp, err := h.svc.GetStatus(c.Request().Context(), fileID)
	finalStatus := "UPLOADED"
	if err == nil {
		finalStatus = statusResp.Status
	}

	return c.JSON(http.StatusOK, response.OK(CompleteUploadResponse{
		FileID: fileID,
		Status: finalStatus,
	}, "upload complete"))
}
