package httpapi

// Shopping lists — lists own sections and items; every route first proves
// the list belongs to the caller (sections/items resolve their list_id and
// go through the same check).

import (
	"errors"
	"net/http"

	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

type shoppingListInput struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type shoppingSectionInput struct {
	Name string `json:"name"`
}

type shoppingItemInput struct {
	Name      string  `json:"name"`
	Note      string  `json:"note"`
	Quantity  string  `json:"quantity"`
	SectionID *string `json:"sectionId"`
}

// ownList resolves :id as a list owned by the caller; writes 404 otherwise.
func (s *Server) ownList(c *gin.Context, id string) bool {
	if s.store.ShoppingListOwned(c.Request.Context(), currentUser(c).ID, id) {
		return true
	}
	c.String(http.StatusNotFound, "list not found")
	return false
}

func (s *Server) shoppingLists(c *gin.Context) {
	lists, err := s.store.ListShoppingLists(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list")
		return
	}
	c.JSON(http.StatusOK, lists)
}

func (s *Server) shoppingCreateList(c *gin.Context) {
	var input shoppingListInput
	if !bind(c, &input) || input.Name == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	list, err := s.store.CreateShoppingList(c.Request.Context(), currentUser(c).ID, input.Name, input.Icon)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create list")
		return
	}
	c.JSON(http.StatusCreated, list)
}

func (s *Server) shoppingListDetail(c *gin.Context) {
	id := c.Param("id")
	if !s.ownList(c, id) {
		return
	}
	sections, err := s.store.ListShoppingSections(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list sections")
		return
	}
	items, err := s.store.ListShoppingItems(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list items")
		return
	}
	c.JSON(http.StatusOK, gin.H{"sections": sections, "items": items})
}

func (s *Server) shoppingUpdateList(c *gin.Context) {
	var input shoppingListInput
	if !bind(c, &input) || input.Name == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	list, err := s.store.UpdateShoppingList(c.Request.Context(), currentUser(c).ID, c.Param("id"), input.Name, input.Icon)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "list not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update list")
		return
	}
	c.JSON(http.StatusOK, list)
}

func (s *Server) shoppingDeleteList(c *gin.Context) {
	err := s.store.DeleteShoppingList(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "list not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to delete list")
		return
	}
	c.Status(http.StatusNoContent)
}

// ownSectionList resolves the section's list and proves ownership; returns
// the listID so callers can pass it down to the store.
func (s *Server) ownSectionList(c *gin.Context, sectionID string) (string, bool) {
	listID, err := s.store.ShoppingSectionList(c.Request.Context(), sectionID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !s.store.ShoppingListOwned(c.Request.Context(), currentUser(c).ID, listID)) {
		c.String(http.StatusNotFound, "section not found")
		return "", false
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return "", false
	}
	return listID, true
}

func (s *Server) shoppingCreateSection(c *gin.Context) {
	listID := c.Param("id")
	var input shoppingSectionInput
	if !bind(c, &input) || input.Name == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	if !s.ownList(c, listID) {
		return
	}
	sc, err := s.store.CreateShoppingSection(c.Request.Context(), listID, input.Name)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create section")
		return
	}
	c.JSON(http.StatusCreated, sc)
}

func (s *Server) shoppingUpdateSection(c *gin.Context) {
	var input shoppingSectionInput
	if !bind(c, &input) || input.Name == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	listID, ok := s.ownSectionList(c, c.Param("id"))
	if !ok {
		return
	}
	err := s.store.UpdateShoppingSection(c.Request.Context(), listID, c.Param("id"), input.Name)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "section not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update section")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) shoppingDeleteSection(c *gin.Context) {
	listID, ok := s.ownSectionList(c, c.Param("id"))
	if !ok {
		return
	}
	err := s.store.DeleteShoppingSection(c.Request.Context(), listID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "section not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to delete section")
		return
	}
	c.Status(http.StatusNoContent)
}

// shoppingCheckSection bulk-checks/unchecks every item in a section.
func (s *Server) shoppingCheckSection(c *gin.Context) {
	var input struct {
		Checked *bool `json:"checked"`
	}
	if !bind(c, &input) || input.Checked == nil {
		c.String(http.StatusBadRequest, "checked required")
		return
	}
	listID, ok := s.ownSectionList(c, c.Param("id"))
	if !ok {
		return
	}
	if err := s.store.SetSectionChecked(c.Request.Context(), listID, ptr(c.Param("id")), *input.Checked); err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) shoppingClearPurchased(c *gin.Context) {
	listID := c.Param("id")
	if !s.ownList(c, listID) {
		return
	}
	n, err := s.store.ClearPurchased(c.Request.Context(), listID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": n})
}

func (s *Server) shoppingCreateItem(c *gin.Context) {
	listID := c.Param("id")
	var input shoppingItemInput
	if !bind(c, &input) || input.Name == "" {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	if !s.ownList(c, listID) {
		return
	}
	// A named section must belong to this list — otherwise the item would
	// show up under a section of a different list.
	if input.SectionID != nil && *input.SectionID != "" {
		owner, err := s.store.ShoppingSectionList(c.Request.Context(), *input.SectionID)
		if err != nil || owner != listID {
			c.String(http.StatusBadRequest, "unknown section")
			return
		}
	}
	it, err := s.store.CreateShoppingItem(c.Request.Context(), listID, input.SectionID, input.Name, input.Note, input.Quantity)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to add item")
		return
	}
	c.JSON(http.StatusCreated, it)
}

func (s *Server) shoppingUpdateItem(c *gin.Context) {
	var patch store.ShoppingItemPatch
	if !bind(c, &patch) {
		c.String(http.StatusBadRequest, "invalid patch")
		return
	}
	listID, err := s.store.ShoppingItemList(c.Request.Context(), c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if !s.ownList(c, listID) {
		return
	}
	// Moving into a section: it must live on the same list.
	if patch.SectionID != nil && *patch.SectionID != "" {
		owner, err := s.store.ShoppingSectionList(c.Request.Context(), *patch.SectionID)
		if err != nil || owner != listID {
			c.String(http.StatusBadRequest, "unknown section")
			return
		}
	}
	it, err := s.store.UpdateShoppingItem(c.Request.Context(), listID, c.Param("id"), patch)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update item")
		return
	}
	c.JSON(http.StatusOK, it)
}

func (s *Server) shoppingDeleteItem(c *gin.Context) {
	listID, err := s.store.ShoppingItemList(c.Request.Context(), c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if !s.ownList(c, listID) {
		return
	}
	err = s.store.DeleteShoppingItem(c.Request.Context(), listID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to delete item")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) shoppingSuggest(c *gin.Context) {
	out, err := s.store.ShoppingSuggestions(c.Request.Context(), currentUser(c).ID, c.Query("q"))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, out)
}

func ptr(s string) *string { return &s }
