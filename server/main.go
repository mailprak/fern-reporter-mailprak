// server.go
package main

import (
    "context"
    "log"
    "net"
    "strconv"

    "google.golang.org/grpc"
    "gorm.io/gorm"
    "gorm.io/driver/sqlite"
    pb "fern-reporter/ping"
    gtid "fern-reporter/gettestrunbyid"
    "fern-reporter/pkg/models"
)

type server struct {
    pb.UnimplementedPingServiceServer
}

type servertestbyid struct {
    gtid.UnimplementedTestRunServiceServer
    db *gorm.DB
}

func (s *server) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
    log.Printf("Received: %v", in.GetMessage())
    return &pb.PingResponse{Message: "Pong: " + in.GetMessage()}, nil
}

func (s *servertestbyid) GetTestRunByID(ctx context.Context, req *gtid.GetTestRunByIDRequest) (*gtid.GetTestRunByIDResponse, error) {
    var testRun models.TestRun
    id := req.GetId()
    result := s.db.Where("id = ?", id).First(&testRun)
    if result.Error != nil {
        return nil, result.Error
    }

    response := &gtid.GetTestRunByIDResponse{
        TestRun: &gtid.TestRun{
            Id: strconv.FormatUint(testRun.ID, 10),
            // Add other fields here
        },
    }
    return response, nil
}


func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterPingServiceServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
    // testid starts here
    db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }

    gtid.RegisterTestRunServiceServer(s, &servertestbyid{db: db})
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }

}

