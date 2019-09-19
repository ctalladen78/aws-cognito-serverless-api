package userpool

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/iam"
	"strings"
	"time"
)

type ListUserPoolClientRequest struct {
	PoolID string `json:"pool_id"`
	ClientID string `json:"client_id"`
	Max int64 `json:"max"`
}

type CreateUserPoolClientRequest struct {
	AllowedOAuthFlows []string `json:"allowed_oauth_flows"`
	AllowedOAuthFlowsUserPoolClient bool `json:"allowed_oauth_flows_userpool_client"`
	AllowedOAuthScopes []string `json:"allowed_oauth_scopes"`
	AnalyticsConfiguration AnalyticsConfiguration `json:"analytics_config"`
	CallbackURLs []string `json:"callback_url"`
	ClientName string `json:"client_name"`
	DefaultRedirectURI string `json:"default_redirect_uri"`
	ExplicitAuthFlows []string `json:"explicit_auth_flows"`
	GenerateSecret bool `json:"generate_secret"`
	LogoutURLs []string `json:"logout_urls"`
	ReadAttributes []string `json:"read_attributes"`
	RefreshTokenValidity int64 `json:"refresh_token_validity"`
	SupportedIdentityProviders []string `json:"supported_identity_providers"`
	UserPoolId string `json:"user_pool_id"`
	WriteAttributes []string `json:"write_attributes"`
}

type AnalyticsConfiguration struct {
	ApplicationId string `json:"application_id"`
	ExternalId string `json:"external_id"`
	RoleArn string `json:"role_arn"`
	UserDataShared bool `json:"user_data_shared"`
}

type UserPoolClientResponse struct {
	ResponseCode int `json:"response_code"`
	Message string `json:"message"`
	Client []UserPoolClient `json:"clients"`
}

type UserPoolClient struct {
	ClientID string `json:"client_id"`
	ClientName string `json:"client_name"`
}

type CreatePoolRequest struct {
	EmailMessage string `json:"email_message"`
	EmailSubject string `json:"email_subject"`
	SMSMessage string `json:"sms_message"`
	EmailVerifyMsg string `json:"email_verify_msg"`
	EmailVerifySub string `json:"email_verify_sub"`
	SMSAuthMsg string `json:"sms_auth_msg"`
	SMSVerifyMsg string `json:"sms_verify_msg"`
	PoolName string `json:"pool_name"`
	WaitDays int64 `json:"wait_days"`
}

type PoolItem struct {
	PoolID string `json:"pool_id"`
	PoolName string `json:"pool_name"`
	ClientID string `json:"client_id"`
	CreatedAt time.Time `json:"created_date"`
}

type PoolResponse struct {
	ResponseCode int `json:"response_code"`
	Message string `json:"message"`
	Pools []PoolItem `json:"pools"`
}

type Config struct {
	Information string
	CognitoClient cognitoidentityprovider.CognitoIdentityProvider
	IAMService iam.IAM
}

func getRoleName(poolName string) string {
	name := strings.Replace(poolName, "-", "", -1)
	return name + "-SMS-Role"
}

// Creates Cognito user pool POOL_NAME
func (config Config) CreateUserPool(poolRequest CreatePoolRequest) (response PoolResponse) {
	if len(poolRequest.PoolName) < 2 {
		fmt.Println("Pool name is required")
		response.ResponseCode = 500
		response.Message = fmt.Sprintf("Pool name is required")
		return
	}
	doc := "{ \"Version\": \"2012-10-17\", \"Statement\": [ { \"Sid\": \"\", \"Effect\": \"Allow\", \"Principal\": { \"Service\": \"cognito-idp.amazonaws.com\" }, \"Action\": \"sts:AssumeRole\" } ] }"

	// Create SMS role with pool name, less any hyphens
	roleName := getRoleName(poolRequest.PoolName) // Required
	path := "/service-role/"
	iamResp, iamErr := config.IAMService.CreateRole(
		&iam.CreateRoleInput{
			AssumeRolePolicyDocument: &doc,
			RoleName:                 &roleName,
			Path:                     &path})

	if iamErr != nil {
		fmt.Println("Could not create role")
		response.ResponseCode = 500
		response.Message = fmt.Sprintf("Could not create role %s", iamErr.Error())
		return
	}

	roleArn := iamResp.Role.Arn
	roleID := iamResp.Role.RoleId

	params := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: &poolRequest.PoolName, // Required
		AdminCreateUserConfig: &cognitoidentityprovider.AdminCreateUserConfigType{
			AllowAdminCreateUserOnly: aws.Bool(false), // false == users can sign themselves up
			InviteMessageTemplate: &cognitoidentityprovider.MessageTemplateType{
				EmailMessage: &poolRequest.EmailMessage,     // Welcome message to new users
				EmailSubject: &poolRequest.EmailSubject, // Welcome subject to new users
				SMSMessage:   &poolRequest.SMSMessage,
			},
			UnusedAccountValidityDays: &poolRequest.WaitDays, // How many days to wait before rescinding offer
		},
		AutoVerifiedAttributes: []*string{ // Auto-verified means the user confirmed the SNS message
			aws.String("email"), // Required; either email or phone_number
			aws.String("phone_number"),
		},
		EmailVerificationMessage: &poolRequest.EmailVerifyMsg,
		EmailVerificationSubject: &poolRequest.EmailVerifySub,
		Policies: &cognitoidentityprovider.UserPoolPolicyType{
			PasswordPolicy: &cognitoidentityprovider.PasswordPolicyType{
				MinimumLength:    aws.Int64(6), // Require a password of at least 6 chars
				RequireLowercase: aws.Bool(false),
				RequireNumbers:   aws.Bool(false),
				RequireSymbols:   aws.Bool(false),
				RequireUppercase: aws.Bool(false),
			},
		},
		Schema: []*cognitoidentityprovider.SchemaAttributeType{
			{ // Required
				AttributeDataType:      aws.String("String"),
				DeveloperOnlyAttribute: aws.Bool(false),
				Mutable:                aws.Bool(false),
				Name:                   aws.String("user_name"),
				Required:               aws.Bool(false),
				StringAttributeConstraints: &cognitoidentityprovider.StringAttributeConstraintsType{
					MaxLength: aws.String("64"), // user name can be up to 64 chars
					MinLength: aws.String("3"),  // or as few as 3 chars
				},
			},
		},
		SmsAuthenticationMessage: &poolRequest.SMSAuthMsg,
		SmsConfiguration: &cognitoidentityprovider.SmsConfigurationType{
			SnsCallerArn: roleArn, // Required
			ExternalId:   roleID,
		},
		SmsVerificationMessage: &poolRequest.SMSVerifyMsg,
	}

	cgResp, cgErr := config.CognitoClient.CreateUserPool(params)

	if cgErr != nil {
		fmt.Println("Could not create user pool")
		response.ResponseCode = 500
		response.Message = fmt.Sprintf("Could not create user pool %s", cgErr.Error())
		return
	}

	pool := PoolItem{
		PoolID:    aws.StringValue(cgResp.UserPool.Id),
		PoolName:  aws.StringValue(cgResp.UserPool.Name),
		CreatedAt: aws.TimeValue(cgResp.UserPool.CreationDate),
	}
	response.ResponseCode = 200
	response.Message = "Ok"
	response.Pools = append(response.Pools, pool)
	return
}

func (config Config) ListUserPool(max int64) (response PoolResponse) {
	result, err := config.CognitoClient.ListUserPools(
		&cognitoidentityprovider.ListUserPoolsInput{
			MaxResults: &max,
		}) // .ListBuckets(nil)
	if err != nil {
		fmt.Println("Could not list user pools")
		response.ResponseCode = 500
		response.Message = fmt.Sprintf("Could not list user pools %s", err.Error())
		return
	}

	var poolList []PoolItem
	for _, pool := range result.UserPools {
		pool := PoolItem{
			PoolID:    aws.StringValue(pool.Id),
			PoolName:  aws.StringValue(pool.Name),
			CreatedAt: aws.TimeValue(pool.CreationDate),
		}
		poolList = append(poolList, pool)
	}

	response = PoolResponse{
		ResponseCode: 200,
		Message:      "Ok",
		Pools:        poolList,
	}

	return
}

func (config Config) CreateUserPoolClient (request CreateUserPoolClientRequest) (response UserPoolClientResponse){
	clientInput := &cognitoidentityprovider.CreateUserPoolClientInput{
		AllowedOAuthFlows:               aws.StringSlice(request.AllowedOAuthFlows),
		AllowedOAuthFlowsUserPoolClient: aws.Bool(request.AllowedOAuthFlowsUserPoolClient),
		AllowedOAuthScopes:              aws.StringSlice(request.AllowedOAuthScopes),
		AnalyticsConfiguration:          &cognitoidentityprovider.AnalyticsConfigurationType{
			ApplicationId:  aws.String(request.AnalyticsConfiguration.ApplicationId),
			ExternalId:     aws.String(request.AnalyticsConfiguration.ExternalId),
			RoleArn:        aws.String(request.AnalyticsConfiguration.RoleArn),
			UserDataShared: aws.Bool(request.AnalyticsConfiguration.UserDataShared),
		},
		CallbackURLs:                    aws.StringSlice(request.CallbackURLs),
		ClientName:                      aws.String(request.ClientName),
		DefaultRedirectURI:              aws.String(request.DefaultRedirectURI),
		ExplicitAuthFlows:               aws.StringSlice(request.ExplicitAuthFlows),
		GenerateSecret:                  aws.Bool(request.GenerateSecret),
		LogoutURLs:                      aws.StringSlice(request.LogoutURLs),
		ReadAttributes:                  aws.StringSlice(request.ReadAttributes),
		RefreshTokenValidity:            aws.Int64(request.RefreshTokenValidity),
		SupportedIdentityProviders:      aws.StringSlice(request.SupportedIdentityProviders),
		UserPoolId:                      aws.String(request.UserPoolId),
		WriteAttributes:                 aws.StringSlice(request.WriteAttributes),
	}
	output, err := config.CognitoClient.CreateUserPoolClient(clientInput)
	if err != nil {
		response.Message = err.Error()
		return
	}
	response.Message = "Ok"
	response.Client = append(response.Client, UserPoolClient{
		ClientID:   aws.StringValue(output.UserPoolClient.ClientId),
		ClientName: aws.StringValue(output.UserPoolClient.ClientName),
	})
	return
}

func (config Config) ListUserPoolClients(request *cognitoidentityprovider.ListUserPoolClientsInput) (response UserPoolClientResponse) {
	output, err := config.CognitoClient.ListUserPoolClients(request)
	if err != nil {
		response.ResponseCode = 500
		response.Message = err.Error()
		return
	}


	for _, poolClients := range output.UserPoolClients {
		item := UserPoolClient{
			ClientID:   aws.StringValue(poolClients.ClientId),
			ClientName: aws.StringValue(poolClients.ClientName),
		}
		response.Client = append(response.Client, item)
	}
	response.ResponseCode = 200
	response.Message = "Ok"
	return
}

func (config Config) UpdateUserPoolClient (request CreateUserPoolClientRequest, clientID string) string {
	describeInput := &cognitoidentityprovider.DescribeUserPoolClientInput{
		ClientId:   aws.String(clientID),
		UserPoolId: aws.String(request.UserPoolId),
	}

	response , err := config.CognitoClient.DescribeUserPoolClient(describeInput)
	if err != nil {
		return err.Error()
	}

	updatedInput := &cognitoidentityprovider.UpdateUserPoolClientInput{
		AllowedOAuthFlows:               aws.StringSlice(request.AllowedOAuthFlows),
		AllowedOAuthFlowsUserPoolClient: aws.Bool(request.AllowedOAuthFlowsUserPoolClient),
		AllowedOAuthScopes:              aws.StringSlice(request.AllowedOAuthScopes),
		AnalyticsConfiguration:          &cognitoidentityprovider.AnalyticsConfigurationType{
			ApplicationId:  aws.String(request.AnalyticsConfiguration.ApplicationId),
			ExternalId:     aws.String(request.AnalyticsConfiguration.ExternalId),
			RoleArn:        aws.String(request.AnalyticsConfiguration.RoleArn),
			UserDataShared: aws.Bool(request.AnalyticsConfiguration.UserDataShared),
		},
		CallbackURLs:                    aws.StringSlice(request.CallbackURLs),
		ClientName:                      aws.String(request.ClientName),
		DefaultRedirectURI:              aws.String(request.DefaultRedirectURI),
		ExplicitAuthFlows:               aws.StringSlice(request.ExplicitAuthFlows),
		LogoutURLs:                      aws.StringSlice(request.LogoutURLs),
		ReadAttributes:                  aws.StringSlice(request.ReadAttributes),
		RefreshTokenValidity:            aws.Int64(request.RefreshTokenValidity),
		SupportedIdentityProviders:      aws.StringSlice(request.SupportedIdentityProviders),
		UserPoolId:                      aws.String(request.UserPoolId),
		WriteAttributes:                 aws.StringSlice(request.WriteAttributes),
	}

	config.CognitoClient.UpdateUserPoolClient(updatedInput)
	return response.String()
}

func (config Config) DescribeUserPoolClient (input *cognitoidentityprovider.DescribeUserPoolClientInput) string {
	response , err := config.CognitoClient.DescribeUserPoolClient(input)
	if err != nil {
		return err.Error()
	}
	return response.String()
}

func (config Config) UserPoolClientResponseToJsonString (response UserPoolClientResponse) string {
	responseJson, err := json.Marshal(response)
	if err != nil {
		return "Unable to construct JSON"
	}
	return string(responseJson)
}

func (config Config) PoolResponseToJsonString (response PoolResponse) string {
	responseJson, err := json.Marshal(response)
	if err != nil {
		return "Unable to construct JSON"
	}
	return string(responseJson)
}

func CheckerArrayString(a []string, b []string) []string {
	if a != nil {
		return a
	}
	return b
}

func CheckerString(a string, b string) string {
	if len(a) <= 0 {
		return a
	}
	return b
}

func CheckerInt(a int64, b int64) int64 {
	if a < 0 {
		return a
	}
	return b
}

func CheckerBool(a bool, b bool) bool {
	if a != b {
		return b
	}
	return a
}