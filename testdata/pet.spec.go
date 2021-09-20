// +build ignore

package api

type Error struct {
	Code    int32
	Message string
}

type NewPet struct {
	Name string
	Tag  string
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
	FindPets(params *FindPetsParams) []Pet

	// (POST /pets)
	AddPet(body Pet) Pet

	// (DELETE /pets/{id})
	DeletePet(id int64)

	// (GET /pets)
	FindPetById(id int64) []Pet
}
