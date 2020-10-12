package test

import "os"

func SetEnvs(envs map[string]string) map[string]*string {
	current := map[string]*string{}
	for k,v := range envs {
		env, ok := os.LookupEnv(k)
		current[k] = nil
		if ok {
			current[k] = &env
		}
		err := os.Setenv(k,v)
		if err != nil {
			panic(err)
		}
	}
	return current
}

func ClearEnvs(prevs map[string]*string) {
	for k, v := range prevs {
		if v == nil {
			_ = os.Unsetenv(k)
			continue
		}
		_ = os.Setenv(k,*v)
	}
}