package model

type (
	CredentialService interface {
		GetAllCredentials() map[string]string
		GetKey(name string) string
		SetKey(name, value string) error
		DeleteKey(name string) error
	}

	Credential struct {
		Name  string `db:"name"`
		Value string `db:"value"`
	}
)
