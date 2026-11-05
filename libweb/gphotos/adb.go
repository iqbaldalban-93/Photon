package gphotos

import "app/internal/adbtoken"

func ExtractTokenBindingAliasFromADB(email string) (string, error) {
	return adbtoken.ExtractAlias(email)
}

func AttachTokenBindingAliasFromADB(credentials, alias string) (string, error) {
	return adbtoken.AttachAlias(credentials, alias)
}
