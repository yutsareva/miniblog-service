package utils

import "os"

func GetEnvVar(envVar string) string {
	value, found := os.LookupEnv(envVar)
	if !found {
		panic("Env var '" + envVar + "' not specified")
	}
	return value
}

func GetEnvVarWithDefault(envVar, defaultValue string) string {
	value, found := os.LookupEnv(envVar)
	if !found {
		return defaultValue
	}
	return value
}