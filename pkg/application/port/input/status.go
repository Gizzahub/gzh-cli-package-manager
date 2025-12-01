package input

import (
	"context"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
)

// StatusUseCase defines the interface for status-related operations.
// This is an input port implemented by the application layer.
type StatusUseCase interface {
	// GetStatus retrieves the status of all package managers.
	GetStatus(ctx context.Context, req *dto.StatusRequest) (*dto.StatusResponse, error)
}

// UpdateUseCase defines the interface for update-related operations.
// This is an input port implemented by the application layer.
type UpdateUseCase interface {
	// Update performs update operations on package managers.
	Update(ctx context.Context, req *dto.UpdateRequest) (*dto.UpdateResponse, error)
}
