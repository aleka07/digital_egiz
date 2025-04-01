package performance

import (
	"testing"
	"time"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	testutils "github.com/digital-egiz/backend/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTimeseriesDataQueryPerformance tests the performance of time-series data queries
func TestTimeseriesDataQueryPerformance(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Setup test environment
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create required tables
	ts.SetupTestDatabase(&models.Project{}, &models.Twin{}, &models.TimeseriesData{})

	// Create test data
	projectID := createTestProject(t, ts)
	twinID := createTestTwin(t, ts, projectID)

	// Create history service
	historyService := services.NewHistoryService(ts.DB, ts.Logger)

	// Create test time-series data
	featurePath := "temperature"
	createTimeseriesTestData(t, ts, twinID, featurePath, 10000) // 10K data points

	// Test case: Query time-series data with different time ranges
	t.Run("Query performance for different time ranges", func(t *testing.T) {
		// Test with different time ranges
		testCases := []struct {
			name     string
			duration time.Duration
			limit    int
		}{
			{"Last hour", time.Hour, 100},
			{"Last day", 24 * time.Hour, 100},
			{"Last week", 7 * 24 * time.Hour, 100},
			{"Last month", 30 * 24 * time.Hour, 100},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Set time range
				endTime := time.Now()
				startTime := endTime.Add(-tc.duration)

				// Measure query time
				startQuery := time.Now()
				data, err := historyService.GetTimeseriesData(twinID, featurePath, startTime, endTime, tc.limit)
				queryDuration := time.Since(startQuery)

				// Assert query success
				require.NoError(t, err)
				assert.NotNil(t, data)

				// Log performance
				t.Logf("Query time for %s: %v, returned %d data points", tc.name, queryDuration, len(data))

				// Assert reasonable performance (adjust thresholds based on expected performance)
				assert.Less(t, queryDuration, 500*time.Millisecond, "Query should complete in under 500ms")
			})
		}
	})

	// Test case: Query aggregated time-series data
	t.Run("Query performance for aggregated data", func(t *testing.T) {
		// Test with different aggregation intervals
		testCases := []struct {
			name     string
			interval string
			duration time.Duration
		}{
			{"Hourly for a day", "1h", 24 * time.Hour},
			{"Daily for a month", "1d", 30 * 24 * time.Hour},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Set time range
				endTime := time.Now()
				startTime := endTime.Add(-tc.duration)

				// Measure query time
				startQuery := time.Now()
				data, err := historyService.GetAggregatedData(twinID, featurePath, startTime, endTime, tc.interval)
				queryDuration := time.Since(startQuery)

				// Assert query success
				require.NoError(t, err)
				assert.NotNil(t, data)

				// Log performance
				t.Logf("Aggregation query time for %s (%s): %v, returned %d data points",
					tc.name, tc.interval, queryDuration, len(data))

				// Assert reasonable performance (adjust thresholds based on expected performance)
				assert.Less(t, queryDuration, 1*time.Second, "Aggregation query should complete in under 1s")
			})
		}
	})
}

// Helper functions

// createTestProject creates a test project for performance testing
func createTestProject(t *testing.T, ts *testutils.TestSetup) uint {
	project := &models.Project{
		Name:        "Test Project",
		Description: "Project for performance testing",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result := ts.DB.DB.Create(project)
	require.NoError(t, result.Error)
	require.NotZero(t, project.ID)

	return project.ID
}

// createTestTwin creates a test twin for performance testing
func createTestTwin(t *testing.T, ts *testutils.TestSetup, projectID uint) uint {
	twin := &models.Twin{
		Name:        "Test Twin",
		Description: "Twin for performance testing",
		DittoID:     "org.example:performance-twin",
		ProjectID:   projectID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result := ts.DB.DB.Create(twin)
	require.NoError(t, result.Error)
	require.NotZero(t, twin.ID)

	return twin.ID
}

// createTimeseriesTestData generates test time-series data
func createTimeseriesTestData(t *testing.T, ts *testutils.TestSetup, twinID uint, featurePath string, count int) {
	// Create base time (now minus the number of data points in seconds)
	baseTime := time.Now().Add(-time.Duration(count) * time.Second)

	// Create batch of time-series data
	var dataPoints []models.TimeseriesData

	for i := 0; i < count; i++ {
		// Generate data with some variation
		value := 20.0 + float64(i%10) // Temperature between 20-30°C

		// Create time-series data point
		dataPoint := models.TimeseriesData{
			TwinID:      twinID,
			FeaturePath: featurePath,
			Timestamp:   baseTime.Add(time.Duration(i) * time.Second),
			ValueType:   "float",
			ValueFloat:  &value,
			CreatedAt:   time.Now(),
		}

		dataPoints = append(dataPoints, dataPoint)

		// Insert in batches of 1000 to avoid memory issues
		if len(dataPoints) >= 1000 || i == count-1 {
			result := ts.DB.DB.CreateInBatches(dataPoints, 1000)
			require.NoError(t, result.Error)
			dataPoints = nil // Reset batch
		}
	}

	// Verify data was inserted
	var count_result int64
	result := ts.DB.DB.Model(&models.TimeseriesData{}).Where("twin_id = ? AND feature_path = ?", twinID, featurePath).Count(&count_result)
	require.NoError(t, result.Error)
	require.GreaterOrEqual(t, count_result, int64(count*0.99), "At least 99% of data points should be inserted")
}
