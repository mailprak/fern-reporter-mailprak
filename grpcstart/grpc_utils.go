package main

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/guidewire/fern-reporter/grpcfiles/fernreporter_pb"
	"github.com/guidewire/fern-reporter/pkg/models"
)

// Functions needed for converting protobuf TestRun to go TestRun struct
// Convert a protobuf TestRun to a Go TestRun struct
func ConvertProtoToTestRun(protoTestRun *fernreporter_pb.TestRun) models.TestRun {
	return models.TestRun{
		ID:              protoTestRun.Id,
		TestProjectName: protoTestRun.TestProjectName,
		TestSeed:        protoTestRun.TestSeed,
		StartTime:       convertProtoTimestamp(protoTestRun.StartTime),
		EndTime:         convertProtoTimestamp(protoTestRun.EndTime),
		SuiteRuns:       convertProtoSuiteRuns(protoTestRun.SuiteRuns),
	}
}

// Convert a slice of protobuf SuiteRun messages to Go SuiteRun structs
func convertProtoSuiteRuns(protoSuiteRuns []*fernreporter_pb.SuiteRun) []models.SuiteRun {
	var suiteRuns []models.SuiteRun
	for _, protoSuiteRun := range protoSuiteRuns {
		suiteRuns = append(suiteRuns, models.SuiteRun{
			ID:        protoSuiteRun.Id,
			TestRunID: protoSuiteRun.TestRunId,
			SuiteName: protoSuiteRun.SuiteName,
			StartTime: convertProtoTimestamp(protoSuiteRun.StartTime),
			EndTime:   convertProtoTimestamp(protoSuiteRun.EndTime),
			SpecRuns:  convertProtoSpecRuns(protoSuiteRun.SpecRuns),
		})
	}
	return suiteRuns
}

// Convert a slice of protobuf SpecRun messages to Go SpecRun structs
func convertProtoSpecRuns(protoSpecRuns []*fernreporter_pb.SpecRun) []models.SpecRun {
	var specRuns []models.SpecRun
	for _, protoSpecRun := range protoSpecRuns {
		specRuns = append(specRuns, models.SpecRun{
			ID:              protoSpecRun.Id,
			SuiteID:         protoSpecRun.SuiteId,
			SpecDescription: protoSpecRun.SpecDescription,
			Status:          protoSpecRun.Status,
			Message:         protoSpecRun.Message,
			Tags:            convertProtoTags(protoSpecRun.Tags),
			StartTime:       convertProtoTimestamp(protoSpecRun.StartTime),
			EndTime:         convertProtoTimestamp(protoSpecRun.EndTime),
		})
	}
	return specRuns
}

// Convert a slice of protobuf Tag messages to Go Tag structs
func convertProtoTags(protoTags []*fernreporter_pb.Tag) []models.Tag {
	var tags []models.Tag
	for _, protoTag := range protoTags {
		tags = append(tags, models.Tag{
			ID:   protoTag.Id,
			Name: protoTag.Name,
		})
	}
	return tags
}

// Convert a protobuf timestamp to Go's time.Time
func convertProtoTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{} // Return zero value if nil
	}
	return ts.AsTime()
}

// Functions needed for converting go TestRun struct to protobuf TestRun
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

//Functions to convert TestSummary

// Convert Go TestSummary struct to Protobuf TestSummary
func ConvertTestSummaryToProto(testSummary models.TestSummary) *fernreporter_pb.TestSummary {
	return &fernreporter_pb.TestSummary{
		SuiteRunId:           uint64(testSummary.SuiteRunID),
		SuiteName:            testSummary.SuiteName,
		TestProjectName:      testSummary.TestProjectName,
		StartTime:            timestamppb.New(testSummary.StartTime),
		TotalPassedSpecRuns:  testSummary.TotalPassedSpecRuns,
		TotalSkippedSpecRuns: testSummary.TotalSkippedSpecRuns,
		TotalSpecRuns:        testSummary.TotalSpecRuns,
	}
}
