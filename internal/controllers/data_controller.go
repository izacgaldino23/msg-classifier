package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

// DataController serves the data browsing screen: the page, its table and detail
// partials, the edit endpoints and the bulk delete. It is HTTP-only — every
// rule lives in DataService.
type DataController struct {
	service *services.DataService
}

func NewDataController(service *services.DataService) *DataController {
	return &DataController{service: service}
}

// Page handles GET /data.
func (ctrl *DataController) Page(c *gin.Context) {
	views.RenderPage(c, views.DataPage, views.DataPageContent)
}

// Table handles GET /data/table?kind=...&filter=...&search=... — renders the
// table partial for the requested kind.
func (ctrl *DataController) Table(c *gin.Context) {
	ctrl.renderTable(c, c.Query("kind"), c.Query("filter"), c.Query("search"))
}

// Detail handles GET /data/:kind/:id?mode=view|edit — renders the drawer
// panel body for one record.
func (ctrl *DataController) Detail(c *gin.Context) {
	kind, id, ok := ctrl.path(c)
	if !ok {
		return
	}
	mode := c.Query("mode")
	if mode != "edit" {
		mode = "view"
	}
	ctrl.renderDetail(c, kind, id, mode)
}

// Update handles POST /data/:kind/:id — persists the edited record and re-renders
// the detail in view mode.
func (ctrl *DataController) Update(c *gin.Context) {
	kind, id, ok := ctrl.path(c)
	if !ok {
		return
	}
	form := &models.DataForm{}
	if err := c.Bind(form); err != nil {
		views.RenderError(c, http.StatusBadRequest, messages.BadRequest())
		return
	}
	var err error
	switch kind {
	case services.DataKindContact:
		_, err = ctrl.service.UpdateContact(id, form.Name, form.Phone, form.Email)
	case services.DataKindNotes:
		_, err = ctrl.service.UpdateNote(id, form.Content, form.Date, form.Time, form.ItemText, form.ItemDone)
	case services.DataKindTransactions:
		_, err = ctrl.service.UpdateTransaction(id, *form)
	default:
		views.RenderError(c, http.StatusBadRequest, messages.BadKind())
		return
	}
	if err != nil {
		renderDataError(c, err)
		return
	}
	ctrl.renderDetail(c, kind, id, "view")
}

// Delete handles POST /data/delete — removes the selected records and re-renders
// the table the user was on.
func (ctrl *DataController) Delete(c *gin.Context) {
	form := &models.DataDeleteForm{}
	if err := c.Bind(form); err != nil {
		views.RenderError(c, http.StatusBadRequest, messages.BadRequest())
		return
	}
	if len(form.IDs) == 0 {
		ctrl.renderTable(c, form.Kind, form.Filter, form.Search)
		return
	}
	var err error
	switch form.Kind {
	case services.DataKindContact:
		err = ctrl.service.DeleteContacts(form.IDs)
	case services.DataKindNotes:
		err = ctrl.service.DeleteNotes(form.IDs)
	case services.DataKindTransactions:
		err = ctrl.service.DeleteTransactions(form.IDs)
	default:
		views.RenderError(c, http.StatusBadRequest, messages.BadKind())
		return
	}
	if err != nil {
		renderDataError(c, err)
		return
	}
	ctrl.renderTable(c, form.Kind, form.Filter, form.Search)
}

// ItemRow handles GET /data/item-row — a blank to-do row for the editor.
func (ctrl *DataController) ItemRow(c *gin.Context) {
	views.RenderDataItemRow(c)
}

// path validates the :kind and :id path segments. On failure it has already
// answered the request and returns false.
func (ctrl *DataController) path(c *gin.Context) (string, uint, bool) {
	kind := c.Param("kind")
	if kind != services.DataKindContact && kind != services.DataKindNotes && kind != services.DataKindTransactions {
		views.RenderError(c, http.StatusBadRequest, messages.BadKind())
		return "", 0, false
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		views.RenderError(c, http.StatusBadRequest, messages.BadID())
		return "", 0, false
	}
	return kind, uint(id), true
}

// renderTable loads and renders the table for a kind, filter and search term.
// The POST handlers call it with the form values, so the same view the user was
// on comes back after a delete.
func (ctrl *DataController) renderTable(c *gin.Context, kind, filter, search string) {
	if kind != services.DataKindContact && kind != services.DataKindNotes && kind != services.DataKindTransactions {
		views.RenderError(c, http.StatusBadRequest, messages.BadKind())
		return
	}
	if filter == "" {
		filter = "all"
	}
	switch kind {
	case services.DataKindContact:
		contacts, err := ctrl.service.ListContacts(filter, search)
		if err != nil {
			renderDataError(c, err)
			return
		}
		views.RenderContactsTable(c, filter, search, contacts)
	case services.DataKindNotes:
		notes, err := ctrl.service.ListNotes(filter, search)
		if err != nil {
			renderDataError(c, err)
			return
		}
		views.RenderNotesTable(c, filter, search, notes)
	case services.DataKindTransactions:
		transactions, err := ctrl.service.ListTransactions(filter, search)
		if err != nil {
			renderDataError(c, err)
			return
		}
		views.RenderTransactionsTable(c, filter, search, transactions)
	}
}

// renderDetail loads and renders one record's panel body.
func (ctrl *DataController) renderDetail(c *gin.Context, kind string, id uint, mode string) {
	switch kind {
	case services.DataKindContact:
		contact, err := ctrl.service.GetContact(id)
		if err != nil {
			renderDataError(c, err)
			return
		}
		views.RenderDataDetail(c, views.DataDetailData{Kind: kind, Mode: mode, Contact: contact})
	case services.DataKindNotes:
		note, err := ctrl.service.GetNote(id)
		if err != nil {
			renderDataError(c, err)
			return
		}
		views.RenderDataDetail(c, views.DataDetailData{Kind: kind, Mode: mode, Note: note})
	case services.DataKindTransactions:
		transaction, err := ctrl.service.GetTransaction(id)
		if err != nil {
			renderDataError(c, err)
			return
		}
		views.RenderDataDetail(c, views.DataDetailData{Kind: kind, Mode: mode, Transaction: transaction})
	}
}

// renderDataError maps a data-screen service error to the error partial: invalid
// input is 400, a missing record is 404, anything else follows the shared mapper.
func renderDataError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidFilter), errors.Is(err, services.ErrInvalidData):
		views.RenderError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrNotFound):
		views.RenderError(c, http.StatusNotFound, messages.NotFound())
	default:
		renderServiceError(c, err)
	}
}
