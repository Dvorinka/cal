package httpapi

// File attachments: POST /api/files stores under DATA_DIR/uploads/<user>/,
// GET /api/files/<name> serves only the caller's own files. Notes reference
// them as ![name](/api/files/<name>) in markdown.

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxUploadBytes = 20 << 20 // 20 MB

var reSafeName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,80}$`)

func (s *Server) uploadFile(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)
	header, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "file required (max 20 MB)")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if len(ext) > 10 {
		ext = ext[:10]
	}
	// Generated name only — never trust the client-supplied filename for a path.
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	name := hex.EncodeToString(buf) + ext

	dir := filepath.Join(s.dataDir, "uploads", currentUser(c).ID)
	dst := filepath.Join(dir, name)
	// Quota: settings.quota_mb caps the user's upload dir.
	settings, err := s.store.Settings(c.Request.Context(), currentUser(c).ID)
	if err == nil && settings.QuotaMB > 0 {
		if dirSize(dir)+header.Size > int64(settings.QuotaMB)<<20 {
			c.String(http.StatusInsufficientStorage, "storage quota exceeded")
			return
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		c.String(http.StatusInternalServerError, "storage unavailable")
		return
	}
	if err := c.SaveUploadedFile(header, dst); err != nil {
		c.String(http.StatusInternalServerError, "save failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url":      "/api/files/" + name,
		"name":     header.Filename,
		"markdown": "![" + header.Filename + "](/api/files/" + name + ")",
	})
}

// dirSize totals a directory (one level — uploads are flat per user).
func dirSize(dir string) int64 {
	var total int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil {
			total += info.Size()
		}
	}
	return total
}

// storageUsage reports used/limit for the settings UI.
func (s *Server) storageUsage(c *gin.Context) {
	userID := currentUser(c).ID
	settings, err := s.store.Settings(c.Request.Context(), userID)
	limit := int64(500) << 20
	if err == nil && settings.QuotaMB > 0 {
		limit = int64(settings.QuotaMB) << 20
	}
	c.JSON(http.StatusOK, gin.H{
		"usedBytes":  dirSize(filepath.Join(s.dataDir, "uploads", userID)),
		"quotaBytes": limit,
	})
}

func (s *Server) serveFile(c *gin.Context) {
	name := c.Param("name")
	if !reSafeName.MatchString(name) {
		c.String(http.StatusNotFound, "not found")
		return
	}
	path := filepath.Join(s.dataDir, "uploads", currentUser(c).ID, name)
	c.File(path)
}
