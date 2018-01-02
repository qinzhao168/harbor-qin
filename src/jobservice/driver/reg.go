package Driver

import "errors"

var (
	// ErrConflict represents http 409 error
	ErrConflict = errors.New("conflict")
)

type Registry interface {
	CreateProject(ProjectName string,public int) error
}