package models

type (
	CredentialService interface {
		GetAllCredentials() ([]Credential, error)
		GetKey(name string) (string, error)
		SetKey(name, value string) error
		DeleteKey(name string) error
	}

	Credential struct {
		Name  string `db:"name"`
		Value string `db:"value"`
	}
)
