package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/guidewire/fern-reporter/grpcfiles/fernreporter_pb"
	"github.com/guidewire/fern-reporter/pkg/models"
	"github.com/guidewire/fern-reporter/pkg/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type grpcServer struct {
	fernreporter_pb.UnimplementedFernReporterServiceServer
	db *gorm.DB
}

func (s *grpcServer) SendReport(ctx context.Context, req *fernreporter_pb.ReportRequest) (*fernreporter_pb.ReportResponse, error) {
	log.Printf("Received gRPC report: %s", req.Message)
	return &fernreporter_pb.ReportResponse{Status: "Report received successfully"}, nil
}

// Correct method signature (use types from the generated pb package)
func (s *grpcServer) Ping(ctx context.Context, req *fernreporter_pb.PingRequest) (*fernreporter_pb.PingResponse, error) {
	log.Printf("Received message: %s", req.GetMessage())
	return &fernreporter_pb.PingResponse{Message: "Pong"}, nil
}

// Implement ReportTestRunById
func (s *grpcServer) ReportTestRunById(ctx context.Context, req *fernreporter_pb.ReportTestRunByIdRequest) (*fernreporter_pb.ReportTestRunByIdResponse, error) {
	var testRun models.TestRun

	testRunID := req.GetId()

	// Query database with preloading related fields
	s.db.Preload("SuiteRuns.SpecRuns").Where("id = ?", testRunID).First(&testRun)

	// Return response
	return &fernreporter_pb.ReportTestRunByIdResponse{
		ReportHeader: "Report Header", // To Do: Replace with actual header logic
		TestRun:      convertTestRunToProto(testRun),
	}, nil
}

// reporttestrunall
func (s *grpcServer) ReportTestRunAll(ctx context.Context, empty *emptypb.Empty) (*fernreporter_pb.ReportTestRunAllResponse, error) {
	var testRuns []models.TestRun
	s.db.Preload("SuiteRuns.SpecRuns.Tags").Find(&testRuns)

	totalTests, executedTests, passedTests, failedTests := utils.CalculateTestMetrics(testRuns)

	// Convert testRuns to gRPC message format
	var grpcTestRuns []*fernreporter_pb.TestRun
	for _, t := range testRuns {
		grpcTestRuns = append(grpcTestRuns, convertTestRunToProto(t))
	}

	return &fernreporter_pb.ReportTestRunAllResponse{
		ReportHeader:  "Report Header", // To Do: Replace with actual header logic
		TestRuns:      grpcTestRuns,
		TotalTests:    int64(totalTests),
		ExecutedTests: int64(executedTests),
		PassedTests:   int64(passedTests),
		FailedTests:   int64(failedTests),
	}, nil
}

func (s *grpcServer) GetProjectAll(ctx context.Context, empty *emptypb.Empty) (*fernreporter_pb.GetProjectAllResponse, error) {
	var projectNames []string
	s.db.Table("test_runs").
		Distinct("test_project_name").
		Order("test_project_name asc").
		Pluck("test_project_name", &projectNames)
	return &fernreporter_pb.GetProjectAllResponse{
		Projects: projectNames,
	}, nil
}

func (s *grpcServer) GetTestSummary(ctx context.Context, req *fernreporter_pb.GetTestSummaryRequest) (*fernreporter_pb.GetTestSummaryResponse, error) {

	projectName := req.GetProjectName()

	var testSummaries []models.TestSummary
	s.db.Table("test_runs").
		Joins("INNER JOIN suite_runs ON test_runs.id = suite_runs.test_run_id").
		Joins("INNER JOIN spec_runs ON suite_runs.id = spec_runs.suite_id").
		Select(`suite_runs.id AS suite_run_id, 
			suite_runs.suite_name,
            test_runs.test_project_name, 
            test_runs.start_time, 
            COUNT(spec_runs.id) FILTER (WHERE spec_runs.status = 'passed') AS total_passed_spec_runs, 
			COUNT(spec_runs.id) FILTER (WHERE spec_runs.status = 'skipped') AS total_skipped_spec_runs, 
            COUNT(spec_runs.id) AS total_spec_runs`).
		Where("test_runs.test_project_name = ?", projectName).
		Group("suite_runs.id, test_runs.test_project_name, test_runs.start_time").
		Order("test_runs.start_time").
		Scan(&testSummaries)

	// Convert testSummaries to gRPC message format
	var grpcTestSummaries []*fernreporter_pb.TestSummary
	for _, t := range testSummaries {
		grpcTestSummaries = append(grpcTestSummaries, ConvertTestSummaryToProto(t))
	}

	return &fernreporter_pb.GetTestSummaryResponse{
		TestSummaries: grpcTestSummaries,
	}, nil
}

func (s *grpcServer) ReportTestInsights(ctx context.Context, req *fernreporter_pb.ReportTestInsightsRequest) (*fernreporter_pb.ReportTestInsightsResponse, error) {
	projectName := req.GetProjectName()
	startTimeInput := req.GetStartTime()
	endTimeInput := req.GetEndTime()

	startTime := convertProtoTimestamp(startTimeInput)
	endTime := convertProtoTimestamp(endTimeInput)
	if startTime.IsZero() {
		startTime = time.Now().AddDate(-1, 0, 0)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}
	longestTestRuns := GetLongestTestRuns(s.db, projectName, startTime, endTime)
	numTests := len(longestTestRuns)
	if len(longestTestRuns) > 10 {
		longestTestRuns = longestTestRuns[:10] //only send top 10 longest runs to display
	}

	averageDuration := GetAverageDuration(s.db, projectName, startTime, endTime)
	fmt.Printf("longestTestRuns: %v\n", longestTestRuns)
	fmt.Printf("averageDuration: %v\n", averageDuration)

	var grpcLongestTestRuns []*fernreporter_pb.TestRunInsight
	for _, t := range longestTestRuns {
		grpcLongestTestRuns = append(grpcLongestTestRuns, ConvertTestRunInsightToProto(t))
	}

	return &fernreporter_pb.ReportTestInsightsResponse{
		ReportHeader:    "Fern Report", //config.GetHeaderName()
		ProjectName:     projectName,
		StartTime:       startTimeInput,
		EndTime:         endTimeInput,
		AverageDuration: float32(averageDuration),
		LongestTestRuns: grpcLongestTestRuns,
		NumTests:        int64(numTests),
	}, nil

}

// Implement DeleteTestRun
func (s *grpcServer) DeleteTestRun(ctx context.Context, req *fernreporter_pb.DeleteTestRunRequest) (*fernreporter_pb.DeleteTestRunResponse, error) {
	var testRunModel models.TestRun

	testRunModel.ID = req.GetId()

	// Delete operation
	result := s.db.Delete(&testRunModel)
	if result.Error != nil {
		// Database error
		return &fernreporter_pb.DeleteTestRunResponse{
			Status:  fernreporter_pb.Status_FAILURE,
			Message: "Error deleting test run",
		}, nil
	} else if result.RowsAffected == 0 {
		// No rows affected (test run not found)
		return &fernreporter_pb.DeleteTestRunResponse{
			Status:  fernreporter_pb.Status_FAILURE,
			Message: "Test run not found",
		}, nil
	}

	// Success response
	return &fernreporter_pb.DeleteTestRunResponse{
		Status:  fernreporter_pb.Status_SUCCESS,
		Message: "Test run deleted successfully",
		TestRun: convertTestRunToProto(testRunModel),
	}, nil
}

func (s *grpcServer) UpdateTestRun(ctx context.Context, req *fernreporter_pb.UpdateTestRunRequest) (*fernreporter_pb.UpdateTestRunResponse, error) {
	var testRunModel models.TestRun

	testRunProto := req.GetTestRun()

	// Find the TestRun by ID
	if err := s.db.Where("id = ?", testRunProto.GetId()).First(&testRunModel).Error; err != nil {
		return &fernreporter_pb.UpdateTestRunResponse{
			Status: fernreporter_pb.Status_FAILURE, Message: "TestRun not found",
		}, fmt.Errorf("TestRun not found: %v", err)
	}

	// Update the fields of testRun based on the request
	testRunModel = ConvertProtoToTestRun(testRunProto)

	// Save the updated TestRun in the database
	if err := s.db.Save(&testRunModel).Error; err != nil {
		return &fernreporter_pb.UpdateTestRunResponse{
			Status: fernreporter_pb.Status_FAILURE, Message: "Failed to update TestRun",
		}, fmt.Errorf("failed to update TestRun: %v", err)
	}

	// Return success response with updated TestRun
	return &fernreporter_pb.UpdateTestRunResponse{
		Status: fernreporter_pb.Status_SUCCESS, Message: "TestRun updated successfully",
		TestRun: convertTestRunToProto(testRunModel),
	}, nil
}

func (s *grpcServer) GetTestRunByID(ctx context.Context, req *fernreporter_pb.GetTestRunByIDRequest) (*fernreporter_pb.GetTestRunByIDResponse, error) {
	var testRun models.TestRun
	id := req.GetId()
	result := s.db.Where("id = ?", id).First(&testRun)
	if result.Error != nil {
		return nil, result.Error
	}

	response := &fernreporter_pb.GetTestRunByIDResponse{
		TestRun: convertTestRunToProto(testRun),
	}
	return response, nil
}

func (s *grpcServer) GetTestRunAll(ctx context.Context, empty *emptypb.Empty) (*fernreporter_pb.GetTestRunAllResponse, error) {
	var testRuns []models.TestRun
	if err := s.db.Find(&testRuns).Error; err != nil {
		return nil, err
	}

	// Convert testRuns to gRPC message format
	var grpcTestRuns []*fernreporter_pb.TestRun
	for _, t := range testRuns {
		grpcTestRuns = append(grpcTestRuns, convertTestRunToProto(t))
	}

	return &fernreporter_pb.GetTestRunAllResponse{TestRuns: grpcTestRuns}, nil
}

func ProcessTags(db *gorm.DB, testRun *fernreporter_pb.TestRun) (*fernreporter_pb.ProcessTagsResponse, error) {
	// Process the tags as before
	for i, suite := range testRun.SuiteRuns {
		for j, spec := range suite.SpecRuns {
			var processedTags []*fernreporter_pb.Tag // Use pointer slice

			for _, tag := range spec.Tags {
				var existingTag fernreporter_pb.Tag

				// Check if the tag already exists
				result := db.Where("name = ?", tag.Name).First(&existingTag)

				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// If the tag does not exist, create a new one
					newTag := &fernreporter_pb.Tag{Name: tag.Name} // Use pointer directly
					if err := db.Create(newTag).Error; err != nil {
						return nil, err // Return error if tag creation fails
					}
					processedTags = append(processedTags, newTag)
				} else if result.Error != nil {
					// Return error if there is a problem fetching the tag
					return nil, result.Error
				} else {
					// If the tag exists, use the existing tag
					processedTags = append(processedTags, &existingTag) // Take pointer
				}
			}
			// Correctly associate the processed tags with the specific spec run
			testRun.SuiteRuns[i].SpecRuns[j].Tags = processedTags
		}
	}

	return &fernreporter_pb.ProcessTagsResponse{
		ErrorMessage: "Tags processed successfully",
	}, nil
}

// Convert via JSON
func copyTestRun(source *fernreporter_pb.TestRun) (*fernreporter_pb.TestRun, error) {
	jsonBytes, err := json.Marshal(source) // Serialize source
	if err != nil {
		return nil, err
	}

	var target fernreporter_pb.TestRun
	err = json.Unmarshal(jsonBytes, &target) // Deserialize into target
	if err != nil {
		return nil, err
	}

	return &target, nil
}

func (s *grpcServer) CreateTestRun(ctx context.Context, req *fernreporter_pb.CreateTestRunRequest) (*fernreporter_pb.CreateTestRunResponse, error) {
	testRun := req.GetTestRun()

	// Check if it's a new record
	isNewRecord := testRun.GetId() == 0

	// If not a new record, check if it exists
	if !isNewRecord {
		var existingTestRun models.TestRun
		if err := s.db.Where("id = ?", testRun.GetId()).First(&existingTestRun).Error; err != nil {
			return &fernreporter_pb.CreateTestRunResponse{Status: fernreporter_pb.Status_FAILURE, Message: "record not found"}, err
		}
	}
	//To Do: why we are doing this?
	mappedTestRun, err := copyTestRun(testRun)
	if err != nil {
		return nil, err // Handle conversion error
	}

	// Process tags (assuming ProcessTags function exists)
	response, err := ProcessTags(s.db, mappedTestRun)
	if err != nil {
		return &fernreporter_pb.CreateTestRunResponse{
			Status:  fernreporter_pb.Status_FAILURE,
			Message: response.ErrorMessage,
		}, err
	}

	// Save or update the TestRun record
	testRunModel := ConvertProtoToTestRun(testRun)

	if err := s.db.Save(&testRunModel).Error; err != nil {
		return &fernreporter_pb.CreateTestRunResponse{Status: fernreporter_pb.Status_FAILURE, Message: "error saving record"}, err
	}

	// Return the saved test run as part of the response
	return &fernreporter_pb.CreateTestRunResponse{
		Status: fernreporter_pb.Status_SUCCESS, Message: "TestRun created successfully",
		TestRun: convertTestRunToProto(testRunModel),
	}, nil
}

func StartGRPCServer(context context.Context) {
	//	lis, err := net.Listen("tcp", ":50051") // Use the desired gRPC port
	lis, err := net.Listen("tcp", "0.0.0.0:50051")

	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	s := grpc.NewServer()
	fernreporter_pb.RegisterFernReporterServiceServer(s, &grpcServer{db: db})

	//if err := s.Serve(lis); err != nil {
	//	log.Fatalf("failed to serve: %v", err)
	//}

	// Enable reflection for testing
	reflection.Register(s)

	// Run the gRPC server in a goroutine
	go func() {
		log.Println("gRPC server is running on port 50051")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

}
