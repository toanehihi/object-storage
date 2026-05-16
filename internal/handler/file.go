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

type FileDeletedResponse struct {
	FileID uuid.UUID `json:"fileId"`
	Status string    `json:"status"`
}

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.list)
	g.GET("/:fileId", h.metadata)
	g.GET("/:fileId/download", h.download)
	g.DELETE("/:fileId", h.delete)
}

func (h *FileHandler) list(c echo.Context) error {
	ownerID := middleware.UserIDFromContext(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	files, total, err := h.svc.ListFiles(c.Request().Context(), ownerID, int32(pageSize), int32(offset))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list files")
	}

	return c.JSON(http.StatusOK, response.OK(
		response.Paginated(files, int32(page), int32(pageSize), total), "",
	))
}

func (h *FileHandler) metadata(c echo.Context) error {
	ownerID := middleware.UserIDFromContext(c)

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	file, err := h.svc.GetMetadata(c.Request().Context(), fileID, ownerID)
	if errors.Is(err, repository.ErrFileNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get file")
	}

	return c.JSON(http.StatusOK, response.OK(file, ""))
}

func (h *FileHandler) delete(c echo.Context) error {
	ownerID := middleware.UserIDFromContext(c)

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	if err := h.svc.Delete(c.Request().Context(), fileID, ownerID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete file")
	}

	return c.JSON(http.StatusOK, response.OK(FileDeletedResponse{
		FileID: fileID,
		Status: "DELETED",
	}, "file has been deleted"))
}

func (h *FileHandler) download(c echo.Context) error {
	ownerID := middleware.UserIDFromContext(c)

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	resp, err := h.svc.GetDownloadURL(c.Request().Context(), fileID, ownerID)
	if errors.Is(err, repository.ErrFileNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	if errors.Is(err, service.ErrFileNotReady) {
		return echo.NewHTTPError(http.StatusConflict, "file is not ready for download")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate download URL")
	}

	return c.JSON(http.StatusOK, response.OK(resp, "download URL generated"))
}
