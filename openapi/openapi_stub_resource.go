package openapi

// specStubResource is a stub implementation of SpecResource interface which is used for testing purposes
type specStubResource struct {
	name                    string
	host                    string
	path                    string
	shouldIgnore            bool
	schemaDefinition        *specSchemaDefinition
	resourceGetOperation    *specResourceOperation
	resourcePostOperation   *specResourceOperation
	resourcePutOperation    *specResourceOperation
	resourceDeleteOperation *specResourceOperation
	timeouts                *specTimeouts

	parentResourceNames    []string
	parentPropertyNames    []string
	fullParentResourceName string
	funcGetResourcePath    func(parentIDs []string) (string, error)
}

func newSpecStubResource(name, path string, shouldIgnore bool, schemaDefinition *specSchemaDefinition) *specStubResource {
	return newSpecStubResourceWithOperations(name, path, shouldIgnore, schemaDefinition, nil, nil, nil, nil)
}

func newSpecStubResourceWithOperations(name, path string, shouldIgnore bool, schemaDefinition *specSchemaDefinition, resourcePostOperation, resourcePutOperation, resourceGetOperation, resourceDeleteOperation *specResourceOperation) *specStubResource {
	return &specStubResource{
		name:                    name,
		path:                    path,
		schemaDefinition:        schemaDefinition,
		shouldIgnore:            shouldIgnore,
		resourcePostOperation:   resourcePostOperation,
		resourceGetOperation:    resourceGetOperation,
		resourceDeleteOperation: resourceDeleteOperation,
		resourcePutOperation:    resourcePutOperation,
		timeouts:                &specTimeouts{},
	}
}

func (s *specStubResource) getResourceName() string { return s.name }

func (s *specStubResource) getResourcePath(parentIDs []string) (string, error) {
	if s.funcGetResourcePath != nil {
		return s.funcGetResourcePath(parentIDs)
	}
	return s.path, nil
}

func (s *specStubResource) getResourceSchema() (*specSchemaDefinition, error) {
	return s.schemaDefinition, nil
}

func (s *specStubResource) shouldIgnoreResource() bool { return s.shouldIgnore }

func (s *specStubResource) getResourceOperations() specResourceOperations {
	return specResourceOperations{
		Post:   s.resourcePostOperation,
		Get:    s.resourceGetOperation,
		Put:    s.resourcePutOperation,
		Delete: s.resourceDeleteOperation,
	}
}

func (s *specStubResource) getTimeouts() (*specTimeouts, error) {
	return s.timeouts, nil
}

func (s *specStubResource) getHost() (string, error) {
	return s.host, nil
}

func (s *specStubResource) isSubResource() (bool, []string, string) {
	if len(s.parentResourceNames) > 0 && s.fullParentResourceName != "" {
		return true, s.parentResourceNames, s.fullParentResourceName
	}
	return false, []string{}, ""
}

func (s *specStubResource) getParentPropertiesNames() ([]string, error) {
	if len(s.parentPropertyNames) > 0 {
		return s.parentPropertyNames, nil
	}
	return []string{}, nil //untested
}
