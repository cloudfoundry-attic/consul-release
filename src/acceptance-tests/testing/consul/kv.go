package consul

type KV interface {
	Set(key, value string) error
	Get(key string) (value string, err error)
}
