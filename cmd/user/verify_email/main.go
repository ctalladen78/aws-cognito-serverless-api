package main

import (
	"fmt"
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
	}else {
		cognitoClient = cognitoidentityprovider.New(sessions)
		iamSvc = iam.New(sessions)
	}
}

func EventHandler(event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
	listAttributes := []*cognitoidentityprovider.AttributeType{
		{
			Name:  aws.String("email_verified"),
			Value: aws.String("true"),
		},
	}

	params := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserAttributes: listAttributes,
		UserPoolId:     aws.String(event.UserPoolID),
		Username:       aws.String(event.UserName),
	}

	_, err := cognitoClient.AdminUpdateUserAttributes(params)
	if err != nil {
		return event, err
	}

	fmt.Println("Test -> ", event.UserName)
	return event, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lambda.Start(EventHandler)
}