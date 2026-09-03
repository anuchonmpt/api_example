package list

func GetMigrations() []Migratable {
	return []Migratable{
		&CreateAuthSchema{},
		&SeedRoles{},
		&CreateDocuments{},
	}
}
