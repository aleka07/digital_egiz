package repository

import "gorm.io/gorm"

// RepositoryFactory creates and manages all repositories
type RepositoryFactory struct {
	db             *gorm.DB
	userRepo       UserRepository
	projectRepo    ProjectRepository
	twinRepo       TwinRepository
	twinTypeRepo   TwinTypeRepository
	mlRepo         MLRepository
	timeseriesRepo TimeseriesRepository
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(db *gorm.DB) *RepositoryFactory {
	return &RepositoryFactory{
		db: db,
	}
}

// User returns the user repository
func (f *RepositoryFactory) User() UserRepository {
	if f.userRepo == nil {
		f.userRepo = NewUserRepository(f.db)
	}
	return f.userRepo
}

// Project returns the project repository
func (f *RepositoryFactory) Project() ProjectRepository {
	if f.projectRepo == nil {
		f.projectRepo = NewProjectRepository(f.db)
	}
	return f.projectRepo
}

// Twin returns the twin repository
func (f *RepositoryFactory) Twin() TwinRepository {
	if f.twinRepo == nil {
		f.twinRepo = NewTwinRepository(f.db)
	}
	return f.twinRepo
}

// TwinType returns the twin type repository
func (f *RepositoryFactory) TwinType() TwinTypeRepository {
	if f.twinTypeRepo == nil {
		f.twinTypeRepo = NewTwinTypeRepository(f.db)
	}
	return f.twinTypeRepo
}

// ML returns the ML repository
func (f *RepositoryFactory) ML() MLRepository {
	if f.mlRepo == nil {
		f.mlRepo = NewMLRepository(f.db)
	}
	return f.mlRepo
}

// Timeseries returns the time-series repository
func (f *RepositoryFactory) Timeseries() TimeseriesRepository {
	if f.timeseriesRepo == nil {
		f.timeseriesRepo = NewTimeseriesRepository(f.db)
	}
	return f.timeseriesRepo
}
