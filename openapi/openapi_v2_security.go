package openapi

import (
	"fmt"
	"github.com/go-openapi/spec"
)

const extTfAuthenticationSchemeBearer = "x-terraform-authentication-scheme-bearer"

type specV2Security struct {
	SecurityDefinitions spec.SecurityDefinitions
	GlobalSecurity      []map[string][]string
}

// GetAPIKeySecurityDefinitions returns a list of SpecSecurityDefinition after looping through the SecurityDefinitions
// and selecting only the SecurityDefinitions of type apiKey
func (s *specV2Security) GetAPIKeySecurityDefinitions() (*SpecSecurityDefinitions, error) {
	securityDefinitions := &SpecSecurityDefinitions{}
	for secDefName, secDef := range s.SecurityDefinitions {
		if secDef.Type == "apiKey" {

			switch secDef.In {
			case "header":
				if s.isBearerScheme(secDef) {
					*securityDefinitions = append(*securityDefinitions, newAPIKeyHeaderBearerSecurityDefinition(secDefName))
				} else {
					*securityDefinitions = append(*securityDefinitions, newAPIKeyHeaderSecurityDefinition(secDefName, secDef.Name))
				}
			case "query":
				if s.isBearerScheme(secDef) {
					*securityDefinitions = append(*securityDefinitions, newAPIKeyQueryBearerSecurityDefinition(secDefName))
				} else {
					*securityDefinitions = append(*securityDefinitions, newAPIKeyQuerySecurityDefinition(secDefName, secDef.Name))
				}
			default:
				return nil, fmt.Errorf("apiKey In value '%s' not supported, only 'header' and 'query' values are valid", secDef.In)
			}

		}
	}
	return securityDefinitions, nil
}

func (s *specV2Security) isBearerScheme(secDef *spec.SecurityScheme) bool {
	authScheme, enabled := secDef.Extensions.GetBool(extTfAuthenticationSchemeBearer)
	if authScheme && enabled {
		return true
	}
	return false
}

// GetGlobalSecuritySchemes returns a list of SpecSecuritySchemes that have their corresponding SpecSecurityDefinition
func (s *specV2Security) GetGlobalSecuritySchemes() (SpecSecuritySchemes, error) {
	securitySchemes := createSecuritySchemes(s.GlobalSecurity)
	for _, securityScheme := range securitySchemes {
		secDef, err := s.GetAPIKeySecurityDefinitions()
		if err != nil {
			return SpecSecuritySchemes{}, nil
		}
		secDefFound := secDef.findSecurityDefinitionFor(securityScheme.Name)
		if secDefFound == nil {
			return nil, fmt.Errorf("global security scheme '%s' not found or not matching supported 'apiKey' type", securityScheme.Name)
		}
	}
	return securitySchemes, nil
}
