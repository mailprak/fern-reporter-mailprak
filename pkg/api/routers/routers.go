package routers

import (
    "log"
	"context"
	"net/http"
	"time"

	"fern-reporter/pkg/api/handlers"
	"fern-reporter/pkg/db"
	pb "fern-reporter/ping"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func RegisterRouters(router *gin.Engine) {
	// router.GET("/", handlers.Home)
	handler := handlers.NewHandler(db.GetDb())

	// Establish a connection to the gRPC server
	conn, err := grpc.Dial("grpc-server:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
 //   defer conn.Close()

	api := router.Group("/api")
	{
		testRun := api.Group("/testrun/")
		testRun.GET("/", handler.GetTestRunAll)
		testRun.GET("/:id", handler.GetTestRunByID)
		testRun.POST("/", handler.CreateTestRun)
		testRun.PUT("/:id", handler.UpdateTestRun)
		testRun.DELETE("/:id", handler.DeleteTestRun)
	}
	reports := router.Group("/reports/testruns")
	{
		testRunReport := reports.GET("/", handler.ReportTestRunAll)
		testRunReport.GET("/:id", handler.ReportTestRunById)
	}
	// client := pb.NewPingServiceClient(conn)
	// handler = NewHandler(client)
    // router = gin.Default()
    // router.GET("/ping", handler.Ping)

    // Create a new PingService client
    grpcClient := pb.NewPingServiceClient(conn)

	router.GET("/ping", func(c *gin.Context) {
        PingHandler(c, grpcClient)
    })

//	ping := router.Group("/ping")
//	{
//		ping.GET("/", handler.Ping)
//	}

}

// PingHandler handles HTTP requests and uses the gRPC client to make a gRPC call
func PingHandler(c *gin.Context, grpcClient pb.PingServiceClient) {
    // Create a new context with a timeout
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    // Make the gRPC call
    response, err := grpcClient.Ping(ctx, &pb.PingRequest{Message: "Ping"})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Respond to the HTTP request with the gRPC response
    c.JSON(http.StatusOK, gin.H{"response": response.Message})
}
