package domain

import "time"

type File struct {
	Filename  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Path      string
}
