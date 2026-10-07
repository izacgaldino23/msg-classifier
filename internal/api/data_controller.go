package api

import (
	"net/http"

	"msg-classifier/internal/models"
	"msg-classifier/internal/services"

	"github.com/gin-gonic/gin"
)

// DataController is the read-only browse surface. It reuses DataService rather
// than the three use-case services: browsing carries none of their extraction
// logic, and the service is already the filter gate.
type DataController struct {
	data *services.DataService
}

func NewDataController(data *services.DataService) *DataController {
	return &DataController{data: data}
}

// ListContacts returns the contacts matching the filter, newest first.
//
//	@Summary	List contacts
//	@Tags		contacts
//	@Produce	json
//	@Param		filter	query		string	false	"all|phone|email|name"	(default "all")
//	@Success	200		{array}		models.Contact
//	@Failure	400		{object}	ErrorResponse
//	@Failure	500		{object}	ErrorResponse
//	@Router		/contacts [get]
func (ctrl *DataController) ListContacts(c *gin.Context) {
	contacts, err := ctrl.data.ListContacts(c.Query("filter"))
	if err != nil {
		RenderError(c, StatusFor(err), err.Error())
		return
	}
	if contacts == nil {
		contacts = []models.Contact{}
	}
	c.JSON(http.StatusOK, contacts)
}

// ListNotes returns the notes matching the filter, newest first.
//
//	@Summary	List notes
//	@Tags		notes
//	@Produce	json
//	@Param		filter	query		string	false	"all|note|reminder|todo"	(default "all")
//	@Success	200		{array}		models.Note
//	@Failure	400		{object}	ErrorResponse
//	@Failure	500		{object}	ErrorResponse
//	@Router		/notes [get]
func (ctrl *DataController) ListNotes(c *gin.Context) {
	notes, err := ctrl.data.ListNotes(c.Query("filter"))
	if err != nil {
		RenderError(c, StatusFor(err), err.Error())
		return
	}
	if notes == nil {
		notes = []*models.Note{}
	}
	c.JSON(http.StatusOK, notes)
}

// ListTransactions returns the transactions matching the type and the optional
// search term, newest first.
//
//	@Summary	List transactions
//	@Tags		transactions
//	@Produce	json
//	@Param		filter	query		string	false	"all|compra|venda|pagamento|recebimento|transferencia"	(default "all")
//	@Param		search	query		string	false	"term matched against the party and the original message"
//	@Success	200		{array}		models.Transaction
//	@Failure	400		{object}	ErrorResponse
//	@Failure	500		{object}	ErrorResponse
//	@Router		/transactions [get]
func (ctrl *DataController) ListTransactions(c *gin.Context) {
	transactions, err := ctrl.data.ListTransactions(c.Query("filter"), c.Query("search"))
	if err != nil {
		RenderError(c, StatusFor(err), err.Error())
		return
	}
	if transactions == nil {
		transactions = []*models.Transaction{}
	}
	c.JSON(http.StatusOK, transactions)
}
