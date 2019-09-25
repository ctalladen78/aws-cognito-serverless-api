package user

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

type LoginRequest struct {
	Email string `json:"email_address"`
	Password string `json:"password,omitempty"`
	ClientID string `json:"client_id"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email_address"`
	Password string `json:"password,omitempty"`
	ClientID string `json:"client_id"`
	Code string `json:"confirmation_code"`
}


type UserItem struct {
	UserPoolID string `json:"user_pool_id"`
	User NewUserItem `json:"user"`
}

type NewUserItem struct {
	Name string `json:"name"`
	Email string `json:"email_address"`
	Password string `json:"password, omitempty"`
	EmailVerified string `json:"email_verified"`
	Confirmed string `json:"is_confirmed"`
}

type Config struct {
	Information string
	CognitoClient cognitoidentityprovider.CognitoIdentityProvider
}

type UserResponse struct {
	ResponseCode int `json:"response_code"`
	Message string `json:"message"`
	UserList []NewUserItem `json:"users"`
}

func (config Config) DeleteUser(user UserItem) (string, int) {
	deleteUserInput := &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(user.UserPoolID),
		Username:   aws.String(user.User.Email),
	}
	_ , err := config.CognitoClient.AdminDeleteUser(deleteUserInput)
	if err != nil {
		return err.Error(), 500
	}
	return "Ok", 200
}

func (config Config) AddUser(user UserItem) (response UserResponse) {
	newUser := user.User

	if newUser.Email == "" || user.UserPoolID == "" || newUser.Name == "" {
		fmt.Println("You must supply an email address, user pool ID, and user name")
		fmt.Println("Usage: go run CreateUser.go -e EMAIL-ADDRESS -p USER-POOL-ID -n USER-NAME")
		response.Message = fmt.Sprintf("You must supply an email address, user pool ID, and user name")
		response.ResponseCode = 500
		return
	}

	newUserData := &cognitoidentityprovider.AdminCreateUserInput{
		DesiredDeliveryMediums: []*string{
			aws.String("EMAIL"),
		},
		MessageAction: aws.String("SUPPRESS"),
		UserAttributes: []*cognitoidentityprovider.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(user.User.Email),
			},
			{
				Name:  aws.String("name"),
				Value: aws.String(user.User.Name),
			},
			{
				Name:  aws.String("email_verified"),
				Value: aws.String("true"),
			},
		},
	}

	newUserData.SetUserPoolId(user.UserPoolID)
	newUserData.SetUsername(user.User.Email)

	output, err := config.CognitoClient.AdminCreateUser(newUserData)
	if err != nil {
		response.ResponseCode = 500
		response.Message = fmt.Sprintf("Got error creating user: %s ", err)
		fmt.Println("Got error creating user:", err)
		return
	}

	newUserItem := NewUserItem{
		Name:  "",
		Email: "",
	}
	attributes := output.User.Attributes
	for _, a := range attributes {
		if *a.Name == "name" {
			newUserItem.Name = *a.Value
		} else if *a.Name == "email" {
			newUserItem.Email = *a.Value
		}
	}

	passwordInput := &cognitoidentityprovider.AdminSetUserPasswordInput{
		Password:   aws.String(user.User.Password),
		Permanent:  aws.Bool(true),
		UserPoolId: aws.String(user.UserPoolID),
		Username:  aws.String(user.User.Email),
	}

	_, err = config.CognitoClient.AdminSetUserPassword(passwordInput)
	if err != nil {
		response.ResponseCode = 400
		response.Message = err.Error()
		return
	}

	response = UserResponse{
		ResponseCode: 200,
		Message:      "Ok",
		UserList:     append([]NewUserItem{}, newUserItem),
	}
	return
}

func (config Config) ListUser(item UserItem) (response UserResponse){
	if item.UserPoolID == "" {
		fmt.Println("You must supply a user pool ID")
		response.ResponseCode = 500
		response.Message = "You must supply a user pool ID"
		return
	}

	results, err := config.CognitoClient.ListUsers(
		&cognitoidentityprovider.ListUsersInput{
			UserPoolId: aws.String(item.UserPoolID)})
	if err != nil {
		response.ResponseCode = 500
		response.Message = "Got error listing users"
		fmt.Println("Got error listing users")
		return
	}

	var usersList []NewUserItem

	for _, user := range results.Users {
		newUser := NewUserItem{
			Name:  "",
			Email: "",
		}
		fmt.Println("Users ", user.String())
		attributes := user.Attributes
		for _, a := range attributes {
			if *a.Name == "name" {
				newUser.Name = *a.Value
			} else if *a.Name == "email" {
				newUser.Email = *a.Value
			} else if *a.Name == "email_verified" {
				newUser.EmailVerified = *a.Value
			} else if *a.Name == "is_confirmed" {
				newUser.Confirmed = *a.Value
			}
		}
		usersList = append(usersList, newUser)
	}

	response = UserResponse{
		ResponseCode: 200,
		Message:      "Ok",
		UserList:     usersList,
	}

	return
}

func (config Config) AuthenticateUser(request LoginRequest) (string,int) {
	fmt.Println("Object -> ", request)
	params := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: map[string]*string{
			"USERNAME": aws.String(request.Email),
			"PASSWORD": aws.String(request.Password),
		},
		ClientId: aws.String(request.ClientID),
	}
	authResp, err := config.CognitoClient.InitiateAuth(params)
	if err != nil {
		return err.Error() , 400
	}
	fmt.Println("Result ", authResp)
	return authResp.String(), 200
}

func (config Config) ForgotPassword(request LoginRequest) (string, int) {
	input := &cognitoidentityprovider.ForgotPasswordInput{
		ClientId:          aws.String(request.ClientID),
		Username:          aws.String(request.Email),
	}
	output, error := config.CognitoClient.ForgotPassword(input)
	if error != nil {
		return error.Error(), 500
	}

	return output.String(), 200
}

func (config Config) ConfirmForgotPassword(request ForgotPasswordRequest) (string, int) {
	input := &cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         aws.String(request.ClientID),
		ConfirmationCode: aws.String(request.Code),
		Username:         aws.String(request.Email),
		Password: aws.String(request.Password),
	}
	output, error := config.CognitoClient.ConfirmForgotPassword(input)
	if error != nil {
		return error.Error(), 500
	}

	return output.String(), 200
}

func (config Config) ObjectToJsonString (response UserResponse) string {
	responseJson, err := json.Marshal(response)
	if err != nil {
		return "Unable to construct JSON"
	}
	return string(responseJson)
}