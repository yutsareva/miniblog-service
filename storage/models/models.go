package models

type Post interface {
	GetId() string
	GetAuthorId() string
	GetText() string
	GetCreatedAt() string
	GetLastModifiedAt() string
	GetVersion() int64
}
