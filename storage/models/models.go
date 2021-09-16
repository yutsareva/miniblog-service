package models

import "time"

type Post interface {
	Id() string
	AuthorId() string
	Text() string
	CreatedAt() time.Time
}

