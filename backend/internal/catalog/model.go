package catalog

type Category struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type ListResponse struct {
	Items []Category `json:"items"`
}
