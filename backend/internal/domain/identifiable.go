package domain

// Identifiable 接口，用于泛型仓库识别实体 ID
type Identifiable interface {
	GetID() string
}
