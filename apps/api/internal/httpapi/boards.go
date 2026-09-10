package httpapi

// Kanban boards: boards → columns → cards. Cards are ordinary task entries
// with board_id/column_id/position — board work shows up on the calendar and
// in Today automatically, completing a card completes the task.

import (
	"errors"
	"net/http"
	"strings"

	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

func (s *Server) listBoards(c *gin.Context) {
	boards, err := s.store.ListBoards(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, boards)
}

func (s *Server) createBoard(c *gin.Context) {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	if body.Color == "" {
		body.Color = "sky"
	}
	board, err := s.store.CreateBoard(c.Request.Context(), currentUser(c).ID, strings.TrimSpace(body.Name), body.Color)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, board)
}

func (s *Server) deleteBoard(c *gin.Context) {
	if err := s.store.DeleteBoard(c.Request.Context(), currentUser(c).ID, c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// boardView returns columns + cards — the board page's whole state.
func (s *Server) boardView(c *gin.Context) {
	userID, boardID := currentUser(c).ID, c.Param("id")
	if !s.store.BoardExists(c.Request.Context(), userID, boardID) {
		c.String(http.StatusNotFound, "board not found")
		return
	}
	cols, err := s.store.BoardColumns(c.Request.Context(), userID, boardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	cards, err := s.store.BoardCards(c.Request.Context(), userID, boardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"columns": cols, "cards": cards})
}

func (s *Server) createColumn(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	userID, boardID := currentUser(c).ID, c.Param("id")
	cols, err := s.store.BoardColumns(c.Request.Context(), userID, boardID)
	if err != nil || len(cols) == 0 && !s.store.BoardExists(c.Request.Context(), userID, boardID) {
		c.String(http.StatusNotFound, "board not found")
		return
	}
	col, err := s.store.CreateColumn(c.Request.Context(), userID, boardID, strings.TrimSpace(body.Name), len(cols))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, col)
}

func (s *Server) renameColumn(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	if err := s.store.RenameColumn(c.Request.Context(), currentUser(c).ID, c.Param("id"), strings.TrimSpace(body.Name)); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteColumn(c *gin.Context) {
	if err := s.store.DeleteColumn(c.Request.Context(), currentUser(c).ID, c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// moveCard drops a card into a column at a position. Frontend sends the
// fractional position (midpoint between neighbours); a full rebalance pass is
// unnecessary at personal-planner scale.
func (s *Server) moveCard(c *gin.Context) {
	var body struct {
		ColumnID *string  `json:"columnId"`
		Position float64  `json:"position"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	userID, entryID := currentUser(c).ID, c.Param("id")
	entry, err := s.store.Entry(c.Request.Context(), userID, entryID)
	if err != nil {
		c.String(http.StatusNotFound, "card not found")
		return
	}
	boardID := ""
	if entry.BoardID != nil {
		boardID = *entry.BoardID
	}
	if body.ColumnID != nil && !s.store.ColumnInBoard(c.Request.Context(), userID, boardID, *body.ColumnID) {
		c.String(http.StatusBadRequest, "column not on this board")
		return
	}
	if err := s.store.MoveCard(c.Request.Context(), userID, entryID, boardID, body.ColumnID, body.Position); err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}
