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

const Auth = `
auth: basic
apiKey: header,X-XXX
oidc: oidc,https://example.com/.well-known/openid-configuration
oauth2:
  flow: authorizationCode
  authUrl: https://api.example.com/oauth2/authorize
  tokenUrl: https://api.example.com/oauth2/token
  refreshUrl: https://api.example.com/oauth2/refresh
  scopes:
    read_pets: read your pets
    write_pets: modify pets in your account
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
	// tags to filter by
	Tags []string
	// maximum number of results to return
	Limit int32
}

type Interface interface {
	// Returns all pets from the system that the user has access to
	// Nam sed condimentum est. Maecenas tempor sagittis sapien, nec rhoncus sem sagittis sit amet. Aenean at gravida augue, ac iaculis sem. Curabitur odio lorem, ornare eget elementum nec, cursus id lectus. Duis mi turpis, pulvinar ac eros ac, tincidunt varius justo. In hac habitasse platea dictumst. Integer at adipiscing ante, a sagittis ligula. Aenean pharetra tempor ante molestie imperdiet. Vivamus id aliquam diam. Cras quis velit non tortor eleifend sagittis. Praesent at enim pharetra urna volutpat venenatis eget eget mauris. In eleifend fermentum facilisis. Praesent enim enim, gravida ac sodales sed, placerat id erat. Suspendisse lacus dolor, consectetur non augue vel, vehicula interdum libero. Morbi euismod sagittis libero sed lacinia.
	//
	// Sed tempus felis lobortis leo pulvinar rutrum. Nam mattis velit nisl, eu condimentum ligula luctus nec. Phasellus semper velit eget aliquet faucibus. In a mattis elit. Phasellus vel urna viverra, condimentum lorem id, rhoncus nibh. Ut pellentesque posuere elementum. Sed a varius odio. Morbi rhoncus ligula libero, vel eleifend nunc tristique vitae. Fusce et sem dui. Aenean nec scelerisque tortor. Fusce malesuada accumsan magna vel tempus. Quisque mollis felis eu dolor tristique, sit amet auctor felis gravida. Sed libero lorem, molestie sed nisl in, accumsan tempor nisi. Fusce sollicitudin massa ut lacinia mattis. Sed vel eleifend lorem. Pellentesque vitae felis pretium, pulvinar elit eu, euismod sapien.
	//
	// (GET /pets)
	// 200: pet response
	// default: unexpected error
	FindPets(params FindPetsParams) []Pet

	// Creates a new pet in the store. Duplicates are allowed
	//
	// (POST /pets)
	// 200: pet response
	// default: unexpected error
	AddPet(body Pet) NewPet

	// deletes a single pet based on the ID supplied
	//
	// (DELETE /pets/{id})
	// 204: pet deleted
	// default: unexpected error
	DeletePet(id int64)

	// Returns a user based on a single ID, if the user does not have access to the pet
	//
	// (GET /pets/{id})
	// 200: pet response
	// default: unexpected error
	FindPetById(id int64) Pet
}
