package models

// DataForm is the inbound DTO for POST /data/:kind/:id. Both kinds share one
// form: a contact reads the Name/Phone/Email group, a note reads the
// Content/Date/Time group plus the parallel ItemText/ItemDone arrays rendered by
// the to-do editor. A missing item_done value counts as not done.
type DataForm struct {
	Name     string   `json:"name" form:"name"`
	Phone    string   `json:"phone" form:"phone"`
	Email    string   `json:"email" form:"email"`
	Content  string   `json:"content" form:"content"`
	Date     string   `json:"date" form:"date"`
	Time     string   `json:"time" form:"time"`
	ItemText []string `json:"item_text" form:"item_text"`
	ItemDone []string `json:"item_done" form:"item_done"`
}

// DataDeleteForm is the inbound DTO for POST /data/delete (bulk delete). Kind
// and Filter round-trip through hidden inputs so the table can be re-rendered in
// the same view the user was on.
type DataDeleteForm struct {
	Kind   string `json:"kind" form:"kind"`
	Filter string `json:"filter" form:"filter"`
	IDs    []uint `json:"ids" form:"ids"`
}
