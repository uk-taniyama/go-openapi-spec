// +build ignore

package api

const OpenAPISpec = `
info:
  version: 1.0.0
  title: Swagger Petstore
  description: A sample API that uses a petstore as an example to demonstrate features in the OpenAPI 3.0 specification
  termsOfService: http://swagger.io/terms/
  contact:
    name: Swagger API Team
    email: apiteam@swagger.io
    url: http://swagger.io
  license:
    name: Apache 2.0
    url: https://www.apache.org/licenses/LICENSE-2.0.html
`

type Error struct {
	Code    int32
	Message string
}

type NewPet struct {
	Name string `{min:10,max:10}`
	Tag  string `{min:10,max:20,format:"[a-z][A-Z]"}`
}

type Pet struct {
	NewPet
	Id int64
}

type FindPetsParams struct {
	Tags  []string
	Limit int32
}

type Interface interface {
	// (GET /pets)
	FindPets(params FindPetsParams) []Pet

	// (POST /pets)
	AddPet(body Pet) Pet

	// (DELETE /pets/{id})
	DeletePet(id int64)

	// (GET /pets/{id})
	FindPetById(id int64) []Pet
}
