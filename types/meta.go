package types

import "time"

type Emoji struct {
	Downloaded bool   `json:"downloaded"`
	FileName   string `json:"fileName"`
	Emoji      struct {
		Name     string   `json:"name"`
		Category string   `json:"category"`
		Aliases  []string `json:"aliases"`
	} `json:"emoji"`
}

type Meta struct {
	MetaVersion int       `json:"metaVersion"`
	Host        string    `json:"host"`
	ExportedAt  time.Time `json:"exportedAt"`
	Emojis      []Emoji   `json:"emojis"`
}
