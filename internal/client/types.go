package client

type ListResponse[T any] struct {
	Data     []T `json:"data"`
	NextPage *struct {
		Offset string `json:"offset"`
		Path   string `json:"path"`
		URI    string `json:"uri"`
	} `json:"next_page"`
}

type Workspace struct {
	Gid  string `json:"gid"`
	Name string `json:"name"`
}

type Project struct {
	Gid        string  `json:"gid"`
	Name       string  `json:"name"`
	Archived   bool    `json:"archived,omitempty"`
	Color      *string `json:"color,omitempty"`
	CreatedAt  *string `json:"created_at,omitempty"`
	ModifiedAt *string `json:"modified_at,omitempty"`
	Workspace  *struct {
		Gid  string `json:"gid"`
		Name string `json:"name"`
	} `json:"workspace,omitempty"`
}

type User struct {
	Gid   string  `json:"gid"`
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}
