package presenters

type FeatureResource struct {
	JAID
	Enabled bool `json:"enabled"`
}

func (r FeatureResource) GetName() string {
	return "features"
}

func NewFeatureResource(name string, enabled bool) *FeatureResource {
	return &FeatureResource{
		JAID:    NewJAID(name),
		Enabled: enabled,
	}
}
