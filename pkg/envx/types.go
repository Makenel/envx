package envx

import "time"

type Base struct {
	CreatedAt    	time.Time	 	            `json:"created_at"`
    UpdatedAt    	time.Time             		`json:"updated_at"`
}
type Project struct {
	Base

	Name 			string						`json:"name"`
	Description 	string 						`json:"description"`
	Environments	map[string]Environment 		`json:"environments"`
	DefaultEnv   	string                 		`json:"default_env"`
}

type Environment struct {
    Name      		string            			`json:"name"`
    Variables 		map[string]Variable 		`json:"variables"`
}

type Variable struct {
	Base

    Key         	string    					`json:"key"`
    Value       	string    					`json:"value"`
    Description 	string    					`json:"description,omitempty"`
    IsSecret    	bool      					`json:"is_secret"`
}