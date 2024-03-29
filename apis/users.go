package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"social-api/config"
	"social-api/constants"
	"social-api/helper"
	"social-api/models"
	"social-api/validators"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// UserEndpoint : Create a new User
func UserEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	collection := config.Client.Database("test").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if r.Method == http.MethodPost {
		var user models.User
		var token models.JWT
		var err models.Err
		var errors []models.Err

		_ = json.NewDecoder(r.Body).Decode(&user)

		// Validation
		name := user.Name
		if validators.IsEmpty(name) {
			err.Message = "Name is Empty."
			errors = append(errors, err)
		}
		if !validators.IsChar(name) {
			err.Message = "Invalid Characters in Name."
			errors = append(errors, err)
		}

		email := user.Email
		if !validators.IsEmail(email) {
			err.Message = "Invalid Email."
			errors = append(errors, err)
		}

		// See if user exists
		userExists := collection.FindOne(ctx, bson.M{"email": email})
		if userExists.Err() == nil {
			err.Message = "User already exists."
			errors = append(errors, err)
		}

		// Encrypt Password
		password := user.Password
		if validators.IsEmpty(password) {
			err.Message = "Password cannot be empty."
			errors = append(errors, err)
		} else {
			hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
			if e != nil {
				log.Fatal(e)
			}
			user.Password = string([]byte(hash))
		}

		if len(errors) == 0 {
			user.Date = time.Now()
			_, e := collection.InsertOne(ctx, user)
			if e != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			jwtToken, e := helper.GenerateJWT(user.Email)
			if e != nil {
				log.Println("Something Went Wrong: ", e.Error())
				return
			}
			token.Token = jwtToken
			json.NewEncoder(w).Encode(token)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errors)
		}
	}
}

// UserLogin : Loging in user
func UserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	collection := config.Client.Database("test").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	var existingUser models.User
	var token models.JWT
	var err models.Err
	var errors []models.Err

	_ = json.NewDecoder(r.Body).Decode(&user)

	// Validation
	email := user.Email
	if !validators.IsEmail(email) {
		err.Message = "Invalid Email."
		errors = append(errors, err)
	}

	e := collection.FindOne(ctx, bson.D{{"email", email}}).Decode(&existingUser)
	if e != nil {
		err.Message = "Invalid Credentials"
		errors = append(errors, err)
		json.NewEncoder(w).Encode(errors)
		return
	}

	hashedPassword := existingUser.Password
	password := []byte(user.Password)
	byteHash := []byte(hashedPassword)

	passwordMatch := bcrypt.CompareHashAndPassword(byteHash, password)
	if passwordMatch != nil {
		err.Message = "Invalid Credentials"
		errors = append(errors, err)
		json.NewEncoder(w).Encode(errors)
		return
	}

	if len(errors) != 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errors)
		return
	}

	jwtToken, e := helper.GenerateJWT(existingUser.Email)
	if e != nil {
		log.Println("Something Went Wrong: ", e.Error())
		return
	}
	token.Token = jwtToken
	json.NewEncoder(w).Encode(token)
}

// ProtectedRoute : Test
func ProtectedRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	var user models.User
	fmt.Println("Headers2: ", r.Header)
	tokenString := r.Header["X-Auth-Token"][0]
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return constants.JWTKey, nil
	})
	if err != nil {
		log.Println("Error HUA!")
	}
	log.Println("TOKEN YAHAN HAI: ", token)
	// ... error handling

	// do something with decoded claims
	for key, val := range claims {
		fmt.Printf("Key: %v, value: %v\n", key, val)
	}
	json.NewEncoder(w).Encode(user)
}
