package admin

import (
	"fmt"
	"log"

	"github.com/VsenseTechnologies/skf_plc_http_server/application/usecase/validation"
	"github.com/VsenseTechnologies/skf_plc_http_server/domain/entity"
	"github.com/VsenseTechnologies/skf_plc_http_server/domain/repository"
	"github.com/VsenseTechnologies/skf_plc_http_server/domain/service"
	"github.com/VsenseTechnologies/skf_plc_http_server/presentation/model/request"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserUseCase struct {
	DataBaseService *service.DataBaseService
}

func InitCreateUserUseCase(repo repository.DataBaseRepository) *CreateUserUseCase {
	service := service.NewDataBaseService(repo)
	return &CreateUserUseCase{
		DataBaseService: service,
	}
}

func (u *CreateUserUseCase) Execute(request *request.User) (error, int) {

	if request.Label == "" {
		return fmt.Errorf("label cannot be empty"), 1
	}

	if request.Email == "" {
		return fmt.Errorf("email cannot be empty"), 1
	}

	if request.Password == "" {
		return fmt.Errorf("password cannot be empty"), 1
	}

	error := validation.ValidateEmail(request.Email)

	if error != nil {
		return error, 1
	}

	error = validation.ValidatePassword(request.Password)

	if error != nil {
		return error, 1
	}

	//spinning a go routine to generate password hash
	hashedPasswordChannel := make(chan any)
	go func() {
		hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(request.Password), 5)
		if err != nil {
			hashedPasswordChannel <- err
			return
		}

		hashedPasswordChannel <- string(hashedPasswordBytes)
	}()

	isUserEmailExists, error := u.DataBaseService.CheckUserEmailExists(request.Email)

	if error != nil {
		//releasing the go routine
		<-hashedPasswordChannel
		log.Printf("error occurred with database while checking user email exists for user email -> %s", request.Email)
		return fmt.Errorf("error occurred with database"), 2
	}

	if isUserEmailExists {
		//releasing the go routine
		<-hashedPasswordChannel
		return fmt.Errorf("user email already exists"), 1
	}

	userId := uuid.New().String()

	hashedPasswordChannelResult := <-hashedPasswordChannel

	value, ok := hashedPasswordChannelResult.(string)

	if !ok {
		log.Printf("error occurred while generating hashed password of the user email %s", request.Email)
		return fmt.Errorf("error occurred while generating hashed password"), 2
	}

	user := &entity.User{
		UserId:   userId,
		Label:    request.Label,
		Email:    request.Email,
		Password: string(value),
	}

	error = u.DataBaseService.CreateUser(user)

	if error != nil {
		log.Printf("error occurred with database while creating the user having email -> %s", user.Email)
		return fmt.Errorf("error occurred with database"), 2
	}

	return nil, 0
}
