package assets

// NewAssets returns a new wrapper for the binary assets generated by go-bindata
func NewAssets(names func() []string, asset func(string) ([]byte, error)) Assets {
	return Assets{
		Names: names,
		Asset: asset,
	}
}

// Assets a wrapper for the binary assets
type Assets struct {
	Names func() []string
	Asset func(string) ([]byte, error)
}
