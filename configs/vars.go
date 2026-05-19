package configs

import "os"

type Vars struct {
	Profile string
}

func NewVars() Vars {
	profile := os.Getenv("APP_PROFILE")
	if profile == "" {
		profile = "dev"
	}

	return Vars{Profile: profile}
}
