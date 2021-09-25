package models

type Post interface {
	ToJson() []byte
}
