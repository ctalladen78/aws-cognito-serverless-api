package main

import (
	"encoding/json"
	"fmt"
	"fp-apac-cognito-service/internal/userpool"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/iam"
	"log"
	"os"
)

var cognitoClient *cognitoidentityprovider.CognitoIdentityProvider
var iamSvc *iam.IAM

func init() {
	// Initialize a session that the SDK will use to load configuration,
	// credentials, and region from the shared config file. (~/.aws/config).
	region := os.Getenv("AWS_REGION")
	if sessions, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: &region,
		},
		SharedConfigState: session.SharedConfigEnable,
	}); err != nil {
		fmt.Println(fmt.Sprintf("Failed to connect to AWS: %s ", err.Error()))
	} else {
		cognitoClient = cognitoidentityprovider.New(sessions)
		iamSvc = iam.New(sessions)
	}
}

func EventHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	modelConfig := userpool.Config{
		Information:   "List User Handler",
		CognitoClient: *cognitoClient,
		IAMService:    *iamSvc,
	}

	item := userpool.CreatePoolRequest{
		EmailMessage:   "",
		EmailSubject:   "",
		SMSMessage:     "",
		EmailVerifyMsg: "",
		EmailVerifySub: "",
		SMSAuthMsg:     "",
		SMSVerifyMsg:   "",
		PoolName:       "",
		WaitDays:       0,
	}
	json.Unmarshal([]byte(request.Body), &item)
	response := modelConfig.CreateUserPool(item)

	return events.APIGatewayProxyResponse{
		StatusCode:        response.ResponseCode,
		Headers:           nil,
		MultiValueHeaders: nil,
		Body:              modelConfig.PoolResponseToJsonString(response),
		IsBase64Encoded:   false,
	}, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(EventHandler)
}
