package models

// DataForm is the inbound DTO for POST /data/:kind/:id. Every kind shares one
// form: a contact reads the Name/Phone/Email group, a note reads the
// Content/Date/Time group plus the parallel ItemText/ItemDone arrays rendered by
// the to-do editor, and a transaction reads the Type/Amount/Party group.
// A missing item_done value counts as not done.
type DataForm struct {
	Name     string   `json:"name" form:"name"`
	Phone    string   `json:"phone" form:"phone"`
	Email    string   `json:"email" form:"email"`
	Content  string   `json:"content" form:"content"`
	Date     string   `json:"date" form:"date"`
	Time     string   `json:"time" form:"time"`
	Type     string   `json:"type" form:"type"`
	Amount   string   `json:"amount" form:"amount"`
	Party    string   `json:"party" form:"party"`
	ItemText []string `json:"item_text" form:"item_text"`
	ItemDone []string `json:"item_done" form:"item_done"`
}

// DataDeleteForm is the inbound DTO for POST /data/delete (bulk delete). Kind,
// Filter and Search round-trip through hidden inputs so the table can be
// re-rendered in the same view the user was on.
type DataDeleteForm struct {
	Kind   string `json:"kind" form:"kind"`
	Filter string `json:"filter" form:"filter"`
	Search string `json:"search" form:"search"`
	IDs    []uint `json:"ids" form:"ids"`
}
