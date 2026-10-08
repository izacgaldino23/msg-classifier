package messages

// ContactSaved is the summary once a contact is persisted.
func ContactSaved(id uint) string { return format("contact.saved", id) }

// ContactFound is the summary when the require flow matched a contact.
func ContactFound() string { return T("contact.found") }

// ContactNotFound is the summary when nothing matched the search term.
func ContactNotFound(term string) string { return format("contact.not_found", term) }

// ContactNoData is the summary when the message carried no contact field at all.
func ContactNoData() string { return T("contact.no_data") }

// ContactDuplicate names the contact that already exists (DC-008).
func ContactDuplicate(name string) string { return format("contact.duplicate", name) }