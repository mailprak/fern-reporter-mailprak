package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/grpcfiles/fernreporter_pb"
	"github.com/guidewire/fern-reporter/pkg/models"

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

	// Parse ID
	testRunID := req.Id
	//if err != nil {
	//	return nil, fmt.Errorf("invalid ID format")
	//}

	// Query database with preloading related fields
	s.db.Preload("SuiteRuns.SpecRuns").Where("id = ?", testRunID).First(&testRun)

	// Map database model to protobuf
	var pbSuiteRuns []*fernreporter_pb.SuiteRun
	for _, sr := range testRun.SuiteRuns {
		var pbSpecRuns []*fernreporter_pb.SpecRun
		for _, spec := range sr.SpecRuns {
			var pbTags []*fernreporter_pb.Tag
			for _, tag := range spec.Tags {
				pbTags = append(pbTags, &fernreporter_pb.Tag{Name: tag.Name})
			}
			pbSpecRuns = append(pbSpecRuns, &fernreporter_pb.SpecRun{Tags: pbTags})
		}
		pbSuiteRuns = append(pbSuiteRuns, &fernreporter_pb.SuiteRun{SpecRuns: pbSpecRuns})
	}

	// Return response
	return &fernreporter_pb.ReportTestRunByIdResponse{
		ReportHeader: "Report Header", // Replace with actual header logic
		TestRun: &fernreporter_pb.TestRun{
			Id:        strconv.Itoa(testRunID), // Convert ID back to string
			SuiteRuns: pbSuiteRuns,
		},
	}, nil
}

// reporttestrunall
func (s *grpcServer) ReportTestRunAll(ctx context.Context, empty *emptypb.Empty) (*fernreporter_pb.ReportTestRunAllResponse, error) {
	var testRuns []models.TestRun
	s.db.Preload("SuiteRuns.SpecRuns.Tags").Find(&testRuns)

	// Convert database model to protobuf response
	var pbTestRuns []*fernreporter_pb.TestRun
	for _, tr := range testRuns {
		var pbSuiteRuns []*fernreporter_pb.SuiteRun
		for _, sr := range tr.SuiteRuns {
			var pbSpecRuns []*fernreporter_pb.SpecRun
			for _, spec := range sr.SpecRuns {
				var pbTags []*fernreporter_pb.Tag
				for _, tag := range spec.Tags {
					pbTags = append(pbTags, &fernreporter_pb.Tag{Name: tag.Name})
				}
				pbSpecRuns = append(pbSpecRuns, &fernreporter_pb.SpecRun{Tags: pbTags})
			}
			pbSuiteRuns = append(pbSuiteRuns, &fernreporter_pb.SuiteRun{SpecRuns: pbSpecRuns})
		}
		pbTestRuns = append(pbTestRuns, &fernreporter_pb.TestRun{
			Id:        strconv.FormatUint(tr.ID, 10),
			SuiteRuns: pbSuiteRuns,
		})
	}

	return &fernreporter_pb.ReportTestRunAllResponse{
		ReportHeader: config.GetHeaderName(),
		TestRuns:     pbTestRuns,
	}, nil
}

// Implement DeleteTestRun
func (s *grpcServer) DeleteTestRun(ctx context.Context, req *fernreporter_pb.DeleteTestRunRequest) (*fernreporter_pb.DeleteTestRunResponse, error) {
	var testRun models.TestRun

	// Parse ID
	testRunID, err := strconv.Atoi(req.Id)
	if err != nil {
		return &fernreporter_pb.DeleteTestRunResponse{
			Success: false,
			Message: "Invalid ID format",
		}, nil
	}

	testRun.ID = uint64(testRunID)

	// Delete operation
	result := s.db.Delete(&testRun)
	if result.Error != nil {
		// Database error
		return &fernreporter_pb.DeleteTestRunResponse{
			Success: false,
			Message: "Error deleting test run",
		}, nil
	} else if result.RowsAffected == 0 {
		// No rows affected (test run not found)
		return &fernreporter_pb.DeleteTestRunResponse{
			Success: false,
			Message: "Test run not found",
		}, nil
	}

	// Success response
	return &fernreporter_pb.DeleteTestRunResponse{
		Success: true,
		Message: "Test run deleted successfully",
	}, nil
}

func (s *grpcServer) UpdateTestRun(ctx context.Context, req *fernreporter_pb.UpdateTestRunRequest) (*fernreporter_pb.UpdateTestRunResponse, error) {
	var testRun models.TestRun

	// Find the TestRun by ID
	if err := s.db.Where("id = ?", req.GetId()).First(&testRun).Error; err != nil {
		return &fernreporter_pb.UpdateTestRunResponse{
			Success: false,
			Message: "TestRun not found",
		}, fmt.Errorf("TestRun not found: %v", err)
	}

	// Update the fields of testRun based on the request
	testRun.TestProjectName = req.GetName() // Update the necessary fields

	// Save the updated TestRun in the database
	if err := s.db.Save(&testRun).Error; err != nil {
		return &fernreporter_pb.UpdateTestRunResponse{
			Success: false,
			Message: "Failed to update TestRun",
		}, fmt.Errorf("failed to update TestRun: %v", err)
	}

	// Return success response with updated TestRun
	return &fernreporter_pb.UpdateTestRunResponse{
		Success: true,
		Message: "TestRun updated successfully",
		TestRun: &fernreporter_pb.TestRun{
			Id:   strconv.FormatUint(testRun.ID, 10),
			Name: testRun.TestProjectName, // Include other fields as needed
		},
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
func convertTestRun(source *fernreporter_pb.TestRun) (*fernreporter_pb.TestRun, error) {
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
			return &fernreporter_pb.CreateTestRunResponse{Success: false, ErrorMessage: "record not found"}, err
		}
	}

	mappedTestRun, err := convertTestRun(testRun)
	if err != nil {
		return nil, err // Handle conversion error
	}

	// Process tags (assuming ProcessTags function exists)
	response, err := ProcessTags(s.db, mappedTestRun)
	if err != nil {
		return &fernreporter_pb.CreateTestRunResponse{
			Success: false,
			//	ErrorMessage: err.Error(),
			ErrorMessage: response.ErrorMessage,
		}, err
	}

	// Save or update the TestRun record
	testRunModel := models.TestRun{
		ID:   uint64(testRun.GetId()),
		TestProjectName: testRun.GetName(),
		// Map other fields as needed
	}

	if err := s.db.Save(&testRunModel).Error; err != nil {
		return &fernreporter_pb.CreateTestRunResponse{Success: false, ErrorMessage: "error saving record"}, err
	}

	// Return the saved test run as part of the response
	return &fernreporter_pb.CreateTestRunResponse{
		Success: true,
		TestRun: &fernreporter_pb.TestRun{Id: int64(testRunModel.ID), Name: testRunModel.TestProjectName}, // Map other fields
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

// Convert TestRun struct
func convertTestRunToProto(testRun models.TestRun) *fernreporter_pb.TestRun {
	return &fernreporter_pb.TestRun{
		Id:              testRun.ID,
		TestProjectName: testRun.TestProjectName,
		TestSeed:        testRun.TestSeed,
		StartTime:       timestamppb.New(testRun.StartTime),
		EndTime:         timestamppb.New(testRun.EndTime),
		SuiteRuns:       convertSuiteRunsToProto(testRun.SuiteRuns),
	}
}

// Convert a slice of SuiteRun structs
func convertSuiteRunsToProto(suiteRuns []models.SuiteRun) []*fernreporter_pb.SuiteRun {
	var protoSuiteRuns []*fernreporter_pb.SuiteRun
	for _, suiteRun := range suiteRuns {
		protoSuiteRuns = append(protoSuiteRuns, &fernreporter_pb.SuiteRun{
			Id:        suiteRun.ID,
			TestRunId: suiteRun.TestRunID,
			SuiteName: suiteRun.SuiteName,
			StartTime: timestamppb.New(suiteRun.StartTime),
			EndTime:   timestamppb.New(suiteRun.EndTime),
			SpecRuns:  convertSpecRunsToProto(suiteRun.SpecRuns),
		})
	}
	return protoSuiteRuns
}

// Convert a slice of SpecRun structs
func convertSpecRunsToProto(specRuns []models.SpecRun) []*fernreporter_pb.SpecRun {
	var protoSpecRuns []*fernreporter_pb.SpecRun
	for _, specRun := range specRuns {
		protoSpecRuns = append(protoSpecRuns, &fernreporter_pb.SpecRun{
			Id:              specRun.ID,
			SuiteId:         specRun.SuiteID,
			SpecDescription: specRun.SpecDescription,
			Status:          specRun.Status,
			Message:         specRun.Message,
			Tags:            convertTagsToProto(specRun.Tags),
			StartTime:       timestamppb.New(specRun.StartTime),
			EndTime:         timestamppb.New(specRun.EndTime),
		})
	}
	return protoSpecRuns
}

// Convert a slice of Tag structs
func convertTagsToProto(tags []models.Tag) []*fernreporter_pb.Tag {
	var protoTags []*fernreporter_pb.Tag
	for _, tag := range tags {
		protoTags = append(protoTags, &fernreporter_pb.Tag{
			Id:   tag.ID,
			Name: tag.Name,
		})
	}
	return protoTags
}
