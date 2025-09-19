package graph

import "github.com/GOPAL-YADAV-D/Soter/internal/database"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{
	DB *database.DB
}