package models

type Post interface {
	GetId() string
	GetLastModifiedAt() int64
}
