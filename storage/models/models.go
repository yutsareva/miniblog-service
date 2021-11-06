package models

type Post interface {
	GetId() string
	GetVersion() int64
}
