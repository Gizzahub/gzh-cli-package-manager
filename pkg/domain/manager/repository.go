package manager

import "context"

// Repository defines the interface for manager data access.
// This is a domain-level interface that will be implemented in the infrastructure layer.
type Repository interface {
	// FindAll returns all supported package managers for the current platform.
	FindAll(ctx context.Context) ([]*Manager, error)

	// FindInstalled returns only the installed package managers.
	FindInstalled(ctx context.Context) ([]*Manager, error)

	// FindByID returns a specific manager by its ID.
	FindByID(ctx context.Context, id ManagerID) (*Manager, error)

	// Save persists a manager's state.
	Save(ctx context.Context, m *Manager) error

	// Delete removes a manager's persisted state.
	Delete(ctx context.Context, id ManagerID) error
}
