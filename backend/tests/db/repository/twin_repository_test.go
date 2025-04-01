package repository_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/db/repository"
	testutils "github.com/digital-egiz/backend/tests/utils"
	"github.com/stretchr/testify/assert"
)

func TestTwinRepository_CRUD(t *testing.T) {
	// Setup test environment
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create tables for testing
	ts.SetupTestDatabase(&models.Project{}, &models.Twin{}, &models.TwinType{}, &models.ProjectMember{})

	// Create repository
	repo := repository.NewTwinRepository(ts.DB.DB)

	// Test case: Create project and twin type first (required for twin)
	projectRepo := repository.NewProjectRepository(ts.DB.DB)
	twinTypeRepo := repository.NewTwinTypeRepository(ts.DB.DB)

	// Create test project
	project := &models.Project{
		Name:        "Test Project",
		Description: "Test project for twin repository test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := projectRepo.Create(project)
	assert.NoError(t, err)
	assert.NotZero(t, project.ID)

	// Create test twin type
	twinType := &models.TwinType{
		Name:        "Test Twin Type",
		Description: "Test twin type for twin repository test",
		Version:     "1.0",
		SchemaJSON:  models.JSON([]byte("{}")),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = twinTypeRepo.Create(twinType)
	assert.NoError(t, err)
	assert.NotZero(t, twinType.ID)

	// Test case: Create twin
	t.Run("Should create twin with valid data", func(t *testing.T) {
		// Create test twin
		twin := &models.Twin{
			Name:        "Test Twin",
			Description: "Test twin for repository test",
			DittoID:     "org.example:test-twin-1",
			ProjectID:   project.ID,
			TypeID:      twinType.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Create twin
		err := repo.Create(twin)

		// Assert success
		assert.NoError(t, err)
		assert.NotZero(t, twin.ID)
	})

	// Test case: Get twin by ID
	t.Run("Should get twin by ID", func(t *testing.T) {
		// Create another test twin
		twin := &models.Twin{
			Name:        "Test Twin 2",
			Description: "Test twin 2 for repository test",
			DittoID:     "org.example:test-twin-2",
			ProjectID:   project.ID,
			TypeID:      twinType.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Create twin
		err := repo.Create(twin)
		assert.NoError(t, err)
		assert.NotZero(t, twin.ID)

		// Get twin by ID
		retrievedTwin, err := repo.GetByID(twin.ID)

		// Assert success
		assert.NoError(t, err)
		assert.NotNil(t, retrievedTwin)
		assert.Equal(t, twin.ID, retrievedTwin.ID)
		assert.Equal(t, twin.Name, retrievedTwin.Name)
		assert.Equal(t, twin.Description, retrievedTwin.Description)
		assert.Equal(t, twin.DittoID, retrievedTwin.DittoID)
		assert.Equal(t, twin.ProjectID, retrievedTwin.ProjectID)
		assert.Equal(t, twin.TypeID, retrievedTwin.TypeID)
	})

	// Test case: List twins
	t.Run("Should list twins", func(t *testing.T) {
		// Create several test twins
		for i := 0; i < 5; i++ {
			twin := &models.Twin{
				Name:        "Test Twin List",
				Description: "Test twin for list test",
				DittoID:     "org.example:test-twin-list-" + strconv.Itoa(i),
				ProjectID:   project.ID,
				TypeID:      twinType.ID,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			err := repo.Create(twin)
			assert.NoError(t, err)
		}

		// List twins
		twins, total, err := repo.ListByProjectID(project.ID, 0, 10)

		// Assert success
		assert.NoError(t, err)
		assert.NotNil(t, twins)
		assert.GreaterOrEqual(t, len(twins), 7)   // 5 + 2 from previous tests
		assert.Equal(t, total, int64(len(twins))) // Total should match the number of twins
	})

	// Test case: List twins with pagination
	t.Run("Should list twins with pagination", func(t *testing.T) {
		// Create a unique twin for pagination test
		twin := &models.Twin{
			Name:        "Pagination Test Twin",
			Description: "Test twin for pagination test",
			DittoID:     "org.example:pagination-twin",
			ProjectID:   project.ID,
			TypeID:      twinType.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.Create(twin)
		assert.NoError(t, err)

		// List twins with pagination (offset 0, limit 2)
		twins, total, err := repo.ListByProjectID(project.ID, 0, 2)

		// Assert success
		assert.NoError(t, err)
		assert.NotNil(t, twins)
		assert.Len(t, twins, 2)                   // Should return exactly 2 twins
		assert.GreaterOrEqual(t, total, int64(3)) // We have at least 3 twins now
	})

	// Test case: Update twin
	t.Run("Should update twin", func(t *testing.T) {
		// Create test twin
		twin := &models.Twin{
			Name:        "Original Twin",
			Description: "Original description",
			DittoID:     "org.example:update-twin",
			ProjectID:   project.ID,
			TypeID:      twinType.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Create twin
		err := repo.Create(twin)
		assert.NoError(t, err)
		assert.NotZero(t, twin.ID)

		// Update twin
		twin.Name = "Updated Twin"
		twin.Description = "Updated description"

		err = repo.Update(twin)
		assert.NoError(t, err)

		// Get twin to verify update
		retrievedTwin, err := repo.GetByID(twin.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Twin", retrievedTwin.Name)
		assert.Equal(t, "Updated description", retrievedTwin.Description)
	})

	// Test case: Delete twin
	t.Run("Should delete twin", func(t *testing.T) {
		// Create test twin
		twin := &models.Twin{
			Name:        "Twin to Delete",
			Description: "This twin will be deleted",
			DittoID:     "org.example:delete-twin",
			ProjectID:   project.ID,
			TypeID:      twinType.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Create twin
		err := repo.Create(twin)
		assert.NoError(t, err)
		assert.NotZero(t, twin.ID)

		// Delete twin
		err = repo.Delete(twin.ID)
		assert.NoError(t, err)

		// Attempt to get deleted twin
		retrievedTwin, err := repo.GetByID(twin.ID)
		assert.Error(t, err)
		assert.Nil(t, retrievedTwin)
	})
}
